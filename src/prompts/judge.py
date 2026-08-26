JUDGE_PROMPT = """You are a classification judge. A code reviewer was given backend code and a database schema and asked to find data-layer issues. Below you are given the expected findings (ground truth) and the reviewer's actual output. Your job is to determine whether the reviewer's output matches each expected finding.

# Expected findings

{smell_entries}

# Reviewer's output

<reviewer_output>
{model_output}
</reviewer_output>

# Task

For each expected finding above, classify whether the reviewer's output agrees with the expected result. The reviewer may use different terminology — match on the underlying concept, not exact keywords.

Rules:
- For findings where the smell SHOULD be detected: mark "detected": true ONLY if the reviewer clearly describes the same core problem. Terminology may differ, but the concept must match.
- For findings where the smell should NOT be detected: mark "detected": true if the reviewer incorrectly flags this issue despite it not being a real problem. Mark "detected": false if the reviewer correctly does not mention it or explicitly dismisses it.
- Match on the **root cause** of the problem, not on surface-level symptoms. Two issues may describe similar observable effects but have fundamentally different causes — treat them as different.
- Do NOT infer or extrapolate. Only mark as detected if the reviewer explicitly describes the problem.

Respond with ONLY a JSON array. No prose, no markdown fences, no commentary.

{output_schema}"""
