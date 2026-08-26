"""
run_ablation_paraphrase.py — Prompt sensitivity ablation (Definition paraphrasing).

Runs DEF evaluation with paraphrased issue definitions on GPT-5.4 and DeepSeek-V3.1.
Results are saved under data/results_ablation_paraphrase/.

Usage:
    uv run run_ablation_paraphrase.py
"""

import os
from dotenv import load_dotenv

from src.llm import LLMConfig
from src.prompts import instruction as inst
from src.prompts import definitions as defs
import main as pipeline

# ── Paraphrased definitions (P2) ──────────────────────────────────────────────
# Same semantics as originals, different wording and structure.

P2_CASCADE_DELETE = """
Unsafe Delete Cascade (name: "cascade_delete")
**Definition:** A parent record is deleted, and its dependent child records are also
removed — either through explicit application-level DELETE statements or through an
ON DELETE CASCADE constraint. A separate read endpoint that depends on those child
records now returns incomplete or missing data, because the rows it expects have been
permanently removed.
"""

P2_STALE_AGGREGATE = """
Stale Aggregate (name: "stale_aggregate")
**Definition:** A cached or precomputed summary value stored in a parent table becomes
outdated when the underlying detail records it summarizes are modified by a write
endpoint. A read endpoint serves this outdated summary value without recalculating it
from the current source-of-truth rows, producing incorrect results.
"""

P2_EXPOSED_RECORD = """
Exposed Record (name: "exposed_record")
**Definition:** A record's lifecycle status is updated by a write endpoint (e.g.,
soft-deleted, archived, deactivated, or otherwise marked as no longer active), but a
read endpoint does not filter on this updated status and continues to return the record
to clients as if it were still active and visible.
"""

LABEL_TO_P2 = {
    "cascade_delete": P2_CASCADE_DELETE,
    "stale_aggregate": P2_STALE_AGGREGATE,
    "exposed_record": P2_EXPOSED_RECORD,
}

# ── Models to evaluate ────────────────────────────────────────────────────────

MODELS = [
    {
        "model": "openai/gpt-5.4",
        "env_key": "LLM_JUDGE_API_KEY",   # OpenAI key
    },
    {
        "model": "together_ai/moonshotai/Kimi-K2.6",
        "env_key": "LLM_API_KEY",         # Together AI key
    },
]

LABELS = ["cascade_delete", "stale_aggregate", "exposed_record"]
TASKS_JSONL = "data/tasks/scenarios.jsonl"
RESULTS_DIR = "data/results_ablation_paraphrase"


def run_ablation():
    load_dotenv()

    base_url = os.getenv("LLM_BASE_URL")
    judge_config = LLMConfig(
        os.environ["LLM_JUDGE_MODEL"],
        os.environ["LLM_JUDGE_API_KEY"],
        os.getenv("LLM_JUDGE_BASE_URL"),
    )

    # Monkey-patch the LABEL_TO_DEFINITION map in the pipeline module
    # to use paraphrased definitions
    original_defs = dict(pipeline.LABEL_TO_DEFINITION)

    for label in LABELS:
        pipeline.LABEL_TO_DEFINITION[label] = LABEL_TO_P2[label]

    try:
        for model_info in MODELS:
            model = model_info["model"]
            api_key = os.environ[model_info["env_key"]]
            task_config = LLMConfig(model, api_key, base_url)

            for label in LABELS:
                print(f"\n{'='*60}")
                print(f"  ABLATION: {model} / {label} (paraphrased definition)")
                print(f"{'='*60}\n")

                pipeline.main(
                    jsonl_path=TASKS_JSONL,
                    prompt_type="definition",
                    task_llm_config=task_config,
                    judge_llm_config=judge_config,
                    results_dir=RESULTS_DIR,
                    label=label,
                    variant=None,
                    runs=1,
                    max_workers=36,
                    temperature=0.0,
                    reasoning_effort=None,
                )
    finally:
        # Restore original definitions
        pipeline.LABEL_TO_DEFINITION.update(original_defs)


if __name__ == "__main__":
    run_ablation()
