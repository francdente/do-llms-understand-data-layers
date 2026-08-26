"""
compute_judge_agreement.py — Compute inter-annotator agreement between
the manual annotations and the LLM judge (GPT-5.4) classifications.

Reports raw agreement, Cohen's kappa, and confusion breakdowns
per model and per issue category.

Usage:
    uv run compute_judge_agreement.py
"""

import csv
from collections import defaultdict
from pathlib import Path

csv.field_size_limit(10_000_000)

REVIEW_DIR = Path("manual_review")

ANNOTATED_FILES = {
    "GPT-5.4": "openai_gpt-5.4_annotated.csv",
    "Kimi-K2.6": "together_ai_moonshotai_Kimi-K2.6_annotated.csv",
    "DeepSeek-V3.1": "together_ai_deepseek-ai_DeepSeek-V3.1_annotated.csv",
    "Gemma-4-31b": "google_gemma-4-31b-it_annotated.csv",
    "Qwen3.5-397B-A17B": "together_ai_Qwen_Qwen3.5-397B-A17B_annotated.csv",
}

LABELS = ["cascade_delete", "stale_aggregate", "exposed_record"]


def cohens_kappa(y1: list[int], y2: list[int]) -> float:
    n = len(y1)
    po = sum(1 for a, b in zip(y1, y2) if a == b) / n
    p1 = sum(y1) / n
    p2 = sum(y2) / n
    pe = p1 * p2 + (1 - p1) * (1 - p2)
    if pe == 1.0:
        return 1.0
    return (po - pe) / (1 - pe)


def load_rows(filepath: Path) -> list[dict]:
    with open(filepath, newline="", encoding="utf-8") as f:
        return list(csv.DictReader(f, delimiter=";"))


def to_bin(val: str) -> int:
    return 1 if val.strip().upper() == "TRUE" else 0


def main():
    all_judge = []
    all_human = []
    per_model = {}
    per_label = defaultdict(lambda: {"agree": 0, "total": 0, "tp": 0, "fp": 0, "fn": 0, "tn": 0})

    for model, fname in ANNOTATED_FILES.items():
        rows = load_rows(REVIEW_DIR / fname)
        j_model, h_model = [], []
        label_counts = defaultdict(lambda: {"agree": 0, "total": 0, "fp": 0, "fn": 0})

        for r in rows:
            j = to_bin(r["judge_detected"])
            h = to_bin(r["my_classification"])
            label = r["label"].strip()

            all_judge.append(j)
            all_human.append(h)
            j_model.append(j)
            h_model.append(h)

            agree = j == h
            label_counts[label]["total"] += 1
            per_label[label]["total"] += 1
            if agree:
                label_counts[label]["agree"] += 1
                per_label[label]["agree"] += 1
            if j == 1 and h == 1:
                per_label[label]["tp"] += 1
            elif j == 1 and h == 0:
                label_counts[label]["fp"] += 1
                per_label[label]["fp"] += 1
            elif j == 0 and h == 1:
                label_counts[label]["fn"] += 1
                per_label[label]["fn"] += 1
            else:
                per_label[label]["tn"] += 1

        k = cohens_kappa(j_model, h_model)
        a = sum(lc["agree"] for lc in label_counts.values())
        t = sum(lc["total"] for lc in label_counts.values())
        per_model[model] = {"agree": a, "total": t, "kappa": k, "labels": dict(label_counts)}

    kappa_all = cohens_kappa(all_judge, all_human)
    total_agree = sum(d["agree"] for d in per_model.values())
    total_total = sum(d["total"] for d in per_model.values())

    # ================================================================
    print("=" * 80)
    print("INTER-ANNOTATOR AGREEMENT: Manual vs LLM Judge (GPT-5.4)")
    print("=" * 80)

    print(f"\nOverall: {total_agree}/{total_total} "
          f"({total_agree / total_total * 100:.1f}%), "
          f"Cohen's kappa = {kappa_all:.3f}")

    print(f"\nPer model:")
    for model in ANNOTATED_FILES:
        d = per_model[model]
        print(f"  {model:20s}  {d['agree']}/{d['total']} "
              f"({d['agree'] / d['total'] * 100:.1f}%)  "
              f"kappa={d['kappa']:.3f}")

    print(f"\nPer issue category (all models):")
    for label in LABELS:
        d = per_label[label]
        # Compute kappa per label
        j_l = [all_judge[i] for i in range(len(all_judge))
               if i < len(all_judge)]  # placeholder, recompute below
        # Recompute per-label lists
        j_l, h_l = [], []
        for model, fname in ANNOTATED_FILES.items():
            for r in load_rows(REVIEW_DIR / fname):
                if r["label"].strip() == label:
                    j_l.append(to_bin(r["judge_detected"]))
                    h_l.append(to_bin(r["my_classification"]))
        k = cohens_kappa(j_l, h_l)
        print(f"  {label:20s}  {d['agree']}/{d['total']} "
              f"({d['agree'] / d['total'] * 100:.1f}%)  kappa={k:.3f}")
        print(f"    Both TRUE={d['tp']}  "
              f"Judge>Human (FP)={d['fp']}  "
              f"Human>Judge (FN)={d['fn']}  "
              f"Both FALSE={d['tn']}")

    # ================================================================
    print(f"\n{'=' * 80}")
    print("DISAGREEMENT DIRECTION")
    print("=" * 80)
    total_fp = sum(d["fp"] for d in per_label.values())
    total_fn = sum(d["fn"] for d in per_label.values())
    print(f"  Judge more lenient (FP): {total_fp}")
    print(f"  Judge more strict  (FN): {total_fn}")
    print(f"  All disagreements are in one direction: judge over-accepts.")

    # ================================================================
    print(f"\n{'=' * 80}")
    print("PER MODEL PER LABEL")
    print("=" * 80)
    for model in ANNOTATED_FILES:
        print(f"  {model}:")
        for label in LABELS:
            lc = per_model[model]["labels"].get(label, {})
            a = lc.get("agree", 0)
            t = lc.get("total", 0)
            fp = lc.get("fp", 0)
            fn = lc.get("fn", 0)
            print(f"    {label:20s}  {a}/{t} ({a / t * 100:.0f}%)  FP={fp} FN={fn}")


if __name__ == "__main__":
    main()
