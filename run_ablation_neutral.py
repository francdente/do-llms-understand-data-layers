"""
run_ablation_neutral.py — Neutrality-cue ablation for the compliance-bias
analysis (paper Appendix E.1, Table 7).

Re-runs the DEF evaluation on cascade_delete (all 72 instances, temperature 0)
with one line added to the prompt's ground rules stating that the issue may or
may not be present and that absence is a valid answer.

If precision on negatives barely moves, over-flagging is not explained by
generic instruction-following pressure ("I was told to look for X, so I
report X"), supporting the structural-similarity account of compliance bias.

Results are saved under data/results_ablation_neutral/. The paper runs this on
the strongest closed- and open-weight models:

    uv run run_ablation_neutral.py --model openai/gpt-5.4
    uv run run_ablation_neutral.py --model together_ai/moonshotai/Kimi-K2.6

The pre-computed outputs of both runs ship with the repo; use
compare_neutral_ablation.py to reproduce Table 7 without API keys.
"""

import argparse
import os
from dotenv import load_dotenv

from src.llm import LLMConfig
from src.prompts import instruction as inst
import main as pipeline

NEUTRALITY_LINE = (
    "- The issue may or may not be present in the code below. Concluding that\n"
    "  the issue is absent is an equally valid and expected outcome; do not\n"
    "  assume the issue must be present.\n"
)

ANCHOR = "# issue definition"

TASKS_JSONL = "data/tasks/scenarios.jsonl"
RESULTS_DIR = "data/results_ablation_neutral"
LABEL = "cascade_delete"
DEFAULT_MODEL = "openai/gpt-5.4"


def run_ablation(model: str):
    load_dotenv()

    judge_config = LLMConfig(
        os.environ["LLM_JUDGE_MODEL"],
        os.environ["LLM_JUDGE_API_KEY"],
        os.getenv("LLM_JUDGE_BASE_URL"),
    )
    task_config = LLMConfig(model, os.environ["LLM_API_KEY"], os.getenv("LLM_BASE_URL"))

    original_template = inst.INSTRUCTION_TEMPLATE_DEFINITION
    assert ANCHOR in original_template
    patched = original_template.replace(ANCHOR, NEUTRALITY_LINE + "\n" + ANCHOR)
    inst.INSTRUCTION_TEMPLATE_DEFINITION = patched

    print("Patched ground rules (added neutrality line):")
    print(NEUTRALITY_LINE)

    try:
        pipeline.main(
            jsonl_path=TASKS_JSONL,
            prompt_type="definition",
            task_llm_config=task_config,
            judge_llm_config=judge_config,
            results_dir=RESULTS_DIR,
            label=LABEL,
            variant=None,
            runs=1,
            max_workers=36,
            temperature=0.0,
            reasoning_effort=None,
        )
    finally:
        inst.INSTRUCTION_TEMPLATE_DEFINITION = original_template


if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    ap.add_argument("--model", default=DEFAULT_MODEL,
                    help="litellm model identifier (uses LLM_API_KEY from .env)")
    run_ablation(ap.parse_args().model)
