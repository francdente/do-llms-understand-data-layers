import json

from dataclass_wizard import fromlist

from src.core import Label, Scenario
from src.llm import LLMConfig, invoke_llm
from src.utils import strip_json_fences

JUDGE_PROMPT = """You are a classification judge. A code reviewer was given backend code and a database schema and asked to find data-layer issues. Below you are given the expected findings (ground truth) and the reviewer's actual output. Your job is to determine whether the reviewer's output matches each expected finding.

# Expected findings

{expected}

# Reviewer's output

<reviewer_output>
{model_output}
</reviewer_output>

# Task

For each expected finding above, classify whether the reviewer's output agrees with the expected result. The reviewer may use different terminology — match on the underlying concept, not exact keywords.

Rules:
- For findings where the issue SHOULD be detected: mark "detected": true ONLY if the reviewer clearly describes the same core problem. Terminology may differ, but the concept must match.
- For findings where the issue should NOT be detected: mark "detected": true if the reviewer incorrectly flags this issue despite it not being a real problem. Mark "detected": false if the reviewer correctly does not mention it or explicitly dismisses it.
- Match on the **root cause** of the problem, not on surface-level symptoms. Two issues may describe similar observable effects but have fundamentally different causes — treat them as different.
- Do NOT infer or extrapolate. Only mark as detected if the reviewer explicitly describes the problem.

Respond with ONLY a JSON array. No prose, no markdown fences, no commentary.

{output_schema}
"""


def execute_judge(
    scenario: Scenario,
    llm_output: str,
    config: LLMConfig,
) -> list[Label]:
    expected = _build_expected(scenario)
    output_schema = _build_output_schema(scenario)
    prompt = JUDGE_PROMPT.format(
        expected=expected,
        model_output=llm_output,
        output_schema=output_schema,
    )

    llm_response = invoke_llm(prompt, config.model, config.api_key, config.base_url)
    json_list = json.loads(strip_json_fences(llm_response.content or ""))
    return fromlist(Label, json_list)


def _build_expected(scenario: Scenario) -> str:
    return "\n\n".join(
        f"### {label.name}\n"
        f"**Expected:** {'SHOULD be detected' if label.detected else 'should NOT be detected'}\n"
        f"{label.description}"
        for label in sorted(scenario.ground_truth, key=lambda e: e.name)
    )


def _build_output_schema(scenario: Scenario) -> str:
    return (
        "[\n"
        + ",\n".join(
            f'  {{"name": "{label.name}", "detected": true/false, '
            f'"description": "brief justification for your classification"}}'
            for label in sorted(scenario.ground_truth, key=lambda e: e.name)
        )
        + "\n]"
    )
