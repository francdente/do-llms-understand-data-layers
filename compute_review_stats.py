"""
compute_review_stats.py — Compute per-variant breakdowns and DEF JSON exclusion rates.

Addresses review points:
  1. Per-language/framework/ORM breakdowns (W2, Q2)
  2. DEF invalid-JSON exclusion rates + sensitivity analysis (W3, Q1)
"""

import json
from pathlib import Path
from collections import defaultdict
from dataclasses import dataclass, field

RESULTS = Path("data/results")

# Non-reasoning runs only (manually mapped)
RUNS = {
    "GPT-5.4": {
        "free": RESULTS / "openai_gpt-5.4/free/20260512_180041",
        "def": {
            "cascade_delete": RESULTS / "openai_gpt-5.4/definition/20260513_011923",
            "stale_aggregate": RESULTS / "openai_gpt-5.4/definition/20260513_012219",
            "exposed_record": RESULTS / "openai_gpt-5.4/definition/20260513_012254",
        },
    },
    "Kimi-K2.6": {
        "free": RESULTS / "together_ai_moonshotai_Kimi-K2.6/free/20260513_152654",
        "def": {
            "cascade_delete": RESULTS / "together_ai_moonshotai_Kimi-K2.6/definition/20260520_164134",
            "stale_aggregate": RESULTS / "together_ai_moonshotai_Kimi-K2.6/definition/20260513_154823",
            "exposed_record": RESULTS / "together_ai_moonshotai_Kimi-K2.6/definition/20260513_155139",
        },
    },
    "DeepSeek-V3.1": {
        "free": RESULTS / "together_ai_deepseek-ai_DeepSeek-V3.1/free/20260513_160857",
        "def": {
            "cascade_delete": RESULTS / "together_ai_deepseek-ai_DeepSeek-V3.1/definition/20260513_163337",
            "stale_aggregate": RESULTS / "together_ai_deepseek-ai_DeepSeek-V3.1/definition/20260513_163431",
            "exposed_record": RESULTS / "together_ai_deepseek-ai_DeepSeek-V3.1/definition/20260513_163854",
        },
    },
    "Gemma-4-31b": {
        "free": RESULTS / "google_gemma-4-31b-it/free/20260513_142635",
        "def": {
            "cascade_delete": RESULTS / "google_gemma-4-31b-it/definition/20260513_144350",
            "stale_aggregate": RESULTS / "google_gemma-4-31b-it/definition/20260513_145310",
            "exposed_record": RESULTS / "google_gemma-4-31b-it/definition/20260513_145253",
        },
    },
    "Qwen3.5-397B": {
        "free": RESULTS / "together_ai_Qwen_Qwen3.5-397B-A17B/free/20260521_164430",
        "def": {
            "cascade_delete": RESULTS / "together_ai_Qwen_Qwen3.5-397B-A17B/definition/20260521_164353",
            "stale_aggregate": RESULTS / "together_ai_Qwen_Qwen3.5-397B-A17B/definition/20260521_165327",
            "exposed_record": RESULTS / "together_ai_Qwen_Qwen3.5-397B-A17B/definition/20260521_165352",
        },
    },
}

VARIANT_KEYS = [
    "python_flask_rawsql", "python_flask_sqlalchemy",
    "python_fastapi_rawsql", "python_fastapi_sqlalchemy",
    "js_express_rawsql", "js_express_sequelize",
    "js_fastify_rawsql", "js_fastify_sequelize",
    "go_gin_rawsql", "go_gin_gorm",
    "go_fiber_rawsql", "go_fiber_gorm",
]

LABELS = ["cascade_delete", "stale_aggregate", "exposed_record"]


def variant_to_lang(v: str) -> str:
    if v.startswith("python"): return "Python"
    if v.startswith("js"): return "JavaScript"
    if v.startswith("go"): return "Go"
    return "Unknown"


def variant_to_framework(v: str) -> str:
    parts = v.split("_")
    return parts[1]  # flask, fastapi, express, fastify, gin, fiber


def variant_to_access(v: str) -> str:
    parts = v.split("_")
    access = parts[2]
    if access == "rawsql":
        return "Raw SQL"
    return "ORM"


def is_positive_task(task_name: str) -> bool:
    return "_P" in task_name


@dataclass
class Count:
    tp: int = 0
    fp: int = 0
    tn: int = 0
    fn: int = 0

    def add_detail(self, d: dict):
        self.tp += d["tp"]
        self.fp += d["fp"]
        self.tn += d["tn"]
        self.fn += d["fn"]

    @property
    def precision(self) -> float:
        return self.tp / (self.tp + self.fp) if (self.tp + self.fp) > 0 else 0.0

    @property
    def recall(self) -> float:
        return self.tp / (self.tp + self.fn) if (self.tp + self.fn) > 0 else 0.0

    @property
    def f1(self) -> float:
        p, r = self.precision, self.recall
        return 2 * p * r / (p + r) if (p + r) > 0 else 0.0

    @property
    def total(self) -> int:
        return self.tp + self.fp + self.tn + self.fn


def load_details(run_dir: Path) -> list[dict]:
    m = json.loads((run_dir / "metrics.json").read_text())
    # Combine details from all runs
    details = []
    for run in m["runs"]:
        details.extend(run["details"])
    return details


def print_counts(counts: dict[str, Count], indent: str = "  "):
    for key in sorted(counts):
        c = counts[key]
        print(f"{indent}{key:25s}  n={c.total:3d}  P={c.precision:.2f}  R={c.recall:.2f}  F1={c.f1:.2f}")


def main():
    # ================================================================
    # PART 1: Per-variant breakdown
    # ================================================================
    print("=" * 70)
    print("PART 1: PER-VARIANT BREAKDOWNS")
    print("=" * 70)

    for model_name, model_runs in RUNS.items():
        print(f"\n{'='*60}")
        print(f"  {model_name}")
        print(f"{'='*60}")

        for prompt_type in ["free", "def"]:
            print(f"\n  --- {prompt_type.upper()} ---")

            # Collect all details
            all_details = []
            if prompt_type == "free":
                all_details = load_details(model_runs["free"])
            else:
                for label in LABELS:
                    all_details.extend(load_details(model_runs["def"][label]))

            # Group by language
            by_lang = defaultdict(Count)
            by_framework = defaultdict(Count)
            by_access = defaultdict(Count)
            by_label_lang = defaultdict(lambda: defaultdict(Count))
            by_label_access = defaultdict(lambda: defaultdict(Count))

            for d in all_details:
                lang = variant_to_lang(d["variant"])
                fw = variant_to_framework(d["variant"])
                access = variant_to_access(d["variant"])
                label = d["label"]

                by_lang[lang].add_detail(d)
                by_framework[fw].add_detail(d)
                by_access[access].add_detail(d)
                by_label_lang[label][lang].add_detail(d)
                by_label_access[label][access].add_detail(d)

            print(f"\n  By language:")
            print_counts(by_lang, "    ")

            print(f"\n  By framework:")
            print_counts(by_framework, "    ")

            print(f"\n  By data access:")
            print_counts(by_access, "    ")

            print(f"\n  By label × language:")
            for label in LABELS:
                print(f"    {label}:")
                print_counts(by_label_lang[label], "      ")

            print(f"\n  By label × data access:")
            for label in LABELS:
                print(f"    {label}:")
                print_counts(by_label_access[label], "      ")

    # Aggregate across all models
    print(f"\n\n{'='*70}")
    print("AGGREGATE ACROSS ALL MODELS")
    print(f"{'='*70}")

    for prompt_type in ["free", "def"]:
        print(f"\n  --- {prompt_type.upper()} ---")

        all_details = []
        for model_name, model_runs in RUNS.items():
            if prompt_type == "free":
                all_details.extend(load_details(model_runs["free"]))
            else:
                for label in LABELS:
                    all_details.extend(load_details(model_runs["def"][label]))

        by_lang = defaultdict(Count)
        by_access = defaultdict(Count)
        by_label_lang = defaultdict(lambda: defaultdict(Count))
        by_label_access = defaultdict(lambda: defaultdict(Count))

        for d in all_details:
            lang = variant_to_lang(d["variant"])
            access = variant_to_access(d["variant"])
            label = d["label"]
            by_lang[lang].add_detail(d)
            by_access[access].add_detail(d)
            by_label_lang[label][lang].add_detail(d)
            by_label_access[label][access].add_detail(d)

        print(f"\n  By language (all models aggregated):")
        print_counts(by_lang, "    ")

        print(f"\n  By data access (all models aggregated):")
        print_counts(by_access, "    ")

        print(f"\n  By label × language:")
        for label in LABELS:
            print(f"    {label}:")
            print_counts(by_label_lang[label], "      ")

        print(f"\n  By label × data access:")
        for label in LABELS:
            print(f"    {label}:")
            print_counts(by_label_access[label], "      ")

    # ================================================================
    # PART 2: DEF JSON exclusion rates
    # ================================================================
    print(f"\n\n{'='*70}")
    print("PART 2: DEF JSON EXCLUSION RATES")
    print(f"{'='*70}")

    EXPECTED_PER_LABEL = 72  # 6 tasks × 12 variants

    total_expected = 0
    total_actual = 0
    total_missing_positive = 0
    total_missing_negative = 0

    for model_name, model_runs in RUNS.items():
        print(f"\n  {model_name}:")
        model_missing = 0
        for label in LABELS:
            details = load_details(model_runs["def"][label])
            actual = len(details)
            missing = EXPECTED_PER_LABEL - actual

            total_expected += EXPECTED_PER_LABEL
            total_actual += actual

            if missing > 0:
                # Figure out which task/variant combos are missing
                present = {(d["task"], d["variant"]) for d in details}

                # Determine the 6 tasks for this label
                prefix_map = {"cascade_delete": "CD", "stale_aggregate": "SA", "exposed_record": "ER"}
                prefix = prefix_map[label]
                tasks = [f"{prefix}_short_{t}{i}_test" for t in ["P", "N"] for i in [1, 2, 3]]
                expected_pairs = {(t, v) for t in tasks for v in VARIANT_KEYS}
                missing_pairs = expected_pairs - present

                miss_pos = sum(1 for t, v in missing_pairs if is_positive_task(t))
                miss_neg = sum(1 for t, v in missing_pairs if not is_positive_task(t))
                total_missing_positive += miss_pos
                total_missing_negative += miss_neg

                print(f"    {label:20s}  {actual}/{EXPECTED_PER_LABEL}  "
                      f"(missing {missing}: {miss_pos} positive, {miss_neg} negative)")

                # Show which variants are affected
                missing_variants = defaultdict(int)
                missing_tasks = defaultdict(int)
                for t, v in missing_pairs:
                    missing_variants[v] += 1
                    missing_tasks[t] += 1
                print(f"      Missing by variant: {dict(missing_variants)}")
                print(f"      Missing by task:    {dict(missing_tasks)}")
            else:
                print(f"    {label:20s}  {actual}/{EXPECTED_PER_LABEL}  (none missing)")
                model_missing += 0

    print(f"\n  TOTAL: {total_actual}/{total_expected} instances "
          f"({total_expected - total_actual} excluded, "
          f"{(total_expected - total_actual) / total_expected * 100:.1f}%)")

    # ================================================================
    # PART 2b: Sensitivity analysis — treat exclusions as worst-case
    # ================================================================
    print(f"\n\n{'='*70}")
    print("PART 2b: SENSITIVITY ANALYSIS (exclusions as worst-case)")
    print(f"{'='*70}")
    print("Treating missing positives as FN, missing negatives as FP\n")

    for model_name, model_runs in RUNS.items():
        has_missing = False
        for label in LABELS:
            details = load_details(model_runs["def"][label])
            if len(details) < EXPECTED_PER_LABEL:
                has_missing = True
                break
        if not has_missing:
            continue

        print(f"  {model_name}:")
        for label in LABELS:
            details = load_details(model_runs["def"][label])
            actual = len(details)
            missing = EXPECTED_PER_LABEL - actual

            if missing == 0:
                # No change
                c = Count()
                for d in details:
                    c.add_detail(d)
                print(f"    {label:20s}  (no change)  P={c.precision:.2f}  R={c.recall:.2f}  F1={c.f1:.2f}")
                continue

            # Original metrics
            c_orig = Count()
            for d in details:
                c_orig.add_detail(d)

            # Worst-case: add missing
            prefix_map = {"cascade_delete": "CD", "stale_aggregate": "SA", "exposed_record": "ER"}
            prefix = prefix_map[label]
            tasks = [f"{prefix}_short_{t}{i}_test" for t in ["P", "N"] for i in [1, 2, 3]]
            present = {(d["task"], d["variant"]) for d in details}
            expected_pairs = {(t, v) for t in tasks for v in VARIANT_KEYS}
            missing_pairs = expected_pairs - present

            c_worst = Count()
            c_worst.tp = c_orig.tp
            c_worst.fp = c_orig.fp
            c_worst.tn = c_orig.tn
            c_worst.fn = c_orig.fn

            for t, v in missing_pairs:
                if is_positive_task(t):
                    c_worst.fn += 1  # missed a positive
                else:
                    c_worst.fp += 1  # incorrectly treated as flagged

            print(f"    {label:20s}  original: P={c_orig.precision:.2f} R={c_orig.recall:.2f} F1={c_orig.f1:.2f}  "
                  f"(n={c_orig.total})")
            print(f"    {'':20s}  worst:    P={c_worst.precision:.2f} R={c_worst.recall:.2f} F1={c_worst.f1:.2f}  "
                  f"(n={c_worst.total}, +{missing} missing)")


if __name__ == "__main__":
    main()
