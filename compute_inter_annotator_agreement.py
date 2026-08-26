"""
compute_inter_annotator_agreement.py — Compute inter-annotator agreement
between two human annotators (F, D) and the LLM judge on FREE outputs.

Reports raw agreement and Cohen's kappa for all three pairs:
  F vs D (human–human), F vs Judge, D vs Judge.

The inter-annotation data covers 4 models (864 instances).
GPT-5.4 is excluded because only one author annotated it.

Usage:
    uv run compute_inter_annotator_agreement.py
"""

import csv
from collections import defaultdict
from pathlib import Path

csv.field_size_limit(10_000_000)

REVIEW_DIR = Path("manual_review/inter_annotations")

INTER_FILES = {
    "Kimi-K2.6": "annotations-kimi.csv",
    "DeepSeek-V3.1": "annotations-deepseek.csv",
    "Gemma-4-31b": "annotations-gemma.csv",
    "Qwen3.5-397B-A17B": "annotations-qwen.csv",
}

LABELS = ["cascade_delete", "stale_aggregate", "exposed_record"]


def cohens_kappa(y1: list[int], y2: list[int]) -> float:
    n = len(y1)
    if n == 0:
        return 0.0
    po = sum(1 for a, b in zip(y1, y2) if a == b) / n
    p1 = sum(y1) / n
    p2 = sum(y2) / n
    pe = p1 * p2 + (1 - p1) * (1 - p2)
    if pe == 1.0:
        return 1.0
    return (po - pe) / (1 - pe)


def to_bin(val: str) -> int:
    return 1 if val.strip().upper() == "TRUE" else 0


def load_rows(filepath: Path) -> list[dict]:
    with open(filepath, newline="", encoding="utf-8") as f:
        return list(csv.DictReader(f))


def report_pair(name: str, y1: list[int], y2: list[int],
                per_model: dict, per_label: dict, labels_order: list[str]):
    total = len(y1)
    agree = sum(1 for a, b in zip(y1, y2) if a == b)
    kappa = cohens_kappa(y1, y2)

    print(f"\n{'=' * 80}")
    print(f"  {name}")
    print(f"{'=' * 80}")
    print(f"\n  Overall: {agree}/{total} ({agree / total * 100:.1f}%), "
          f"Cohen's kappa = {kappa:.3f}")

    print(f"\n  Per model:")
    for model in INTER_FILES:
        d = per_model[model]
        k = cohens_kappa(d["y1"], d["y2"])
        a = sum(1 for a, b in zip(d["y1"], d["y2"]) if a == b)
        t = len(d["y1"])
        print(f"    {model:22s}  {a}/{t} ({a / t * 100:.1f}%)  kappa={k:.3f}")

    print(f"\n  Per issue category:")
    for label in labels_order:
        d = per_label[label]
        k = cohens_kappa(d["y1"], d["y2"])
        a = sum(1 for a, b in zip(d["y1"], d["y2"]) if a == b)
        t = len(d["y1"])
        # Confusion breakdown
        tp = sum(1 for a, b in zip(d["y1"], d["y2"]) if a == 1 and b == 1)
        fp = sum(1 for a, b in zip(d["y1"], d["y2"]) if a == 0 and b == 1)
        fn = sum(1 for a, b in zip(d["y1"], d["y2"]) if a == 1 and b == 0)
        tn = sum(1 for a, b in zip(d["y1"], d["y2"]) if a == 0 and b == 0)
        print(f"    {label:22s}  {a}/{t} ({a / t * 100:.1f}%)  kappa={k:.3f}")
        print(f"      Both TRUE={tp}  1>2 (FP)={fp}  1<2 (FN)={fn}  Both FALSE={tn}")

    return agree, total, kappa


def main():
    # Collect all annotations
    all_f, all_d, all_j = [], [], []
    pair_data = {
        "F_D": {"per_model": defaultdict(lambda: {"y1": [], "y2": []}),
                "per_label": defaultdict(lambda: {"y1": [], "y2": []})},
        "F_Judge": {"per_model": defaultdict(lambda: {"y1": [], "y2": []}),
                    "per_label": defaultdict(lambda: {"y1": [], "y2": []})},
        "D_Judge": {"per_model": defaultdict(lambda: {"y1": [], "y2": []}),
                    "per_label": defaultdict(lambda: {"y1": [], "y2": []})},
    }

    total_rows = 0
    for model, fname in INTER_FILES.items():
        rows = load_rows(REVIEW_DIR / fname)
        for r in rows:
            f = to_bin(r["F_classification"])
            d = to_bin(r["D_classification"])
            j = to_bin(r["judge_detected"])
            label = r["label"].strip()

            all_f.append(f)
            all_d.append(d)
            all_j.append(j)

            for pair_key, v1, v2 in [("F_D", f, d), ("F_Judge", f, j), ("D_Judge", d, j)]:
                pair_data[pair_key]["per_model"][model]["y1"].append(v1)
                pair_data[pair_key]["per_model"][model]["y2"].append(v2)
                pair_data[pair_key]["per_label"][label]["y1"].append(v1)
                pair_data[pair_key]["per_label"][label]["y2"].append(v2)

            total_rows += 1

    print(f"Total instances: {total_rows} (4 models x 216 = {4 * 216})")

    # Report each pair
    results = {}
    for pair_key, name, y1, y2 in [
        ("F_D", "Author F vs Author D (human–human)", all_f, all_d),
        ("F_Judge", "Author F vs LLM Judge", all_f, all_j),
        ("D_Judge", "Author D vs LLM Judge", all_d, all_j),
    ]:
        agree, total, kappa = report_pair(
            name, y1, y2,
            pair_data[pair_key]["per_model"],
            pair_data[pair_key]["per_label"],
            LABELS,
        )
        results[pair_key] = {"agree": agree, "total": total, "kappa": kappa}

    # Three-way agreement
    three_agree = sum(1 for f, d, j in zip(all_f, all_d, all_j) if f == d == j)
    print(f"\n{'=' * 80}")
    print(f"  THREE-WAY AGREEMENT (F, D, Judge all agree)")
    print(f"{'=' * 80}")
    print(f"\n  Overall: {three_agree}/{total_rows} ({three_agree / total_rows * 100:.1f}%)")

    per_label_three = defaultdict(lambda: {"agree": 0, "total": 0})
    idx = 0
    for model, fname in INTER_FILES.items():
        rows = load_rows(REVIEW_DIR / fname)
        for r in rows:
            f = to_bin(r["F_classification"])
            d = to_bin(r["D_classification"])
            j = to_bin(r["judge_detected"])
            label = r["label"].strip()
            per_label_three[label]["total"] += 1
            if f == d == j:
                per_label_three[label]["agree"] += 1

    print(f"\n  Per issue category:")
    for label in LABELS:
        d = per_label_three[label]
        print(f"    {label:22s}  {d['agree']}/{d['total']} "
              f"({d['agree'] / d['total'] * 100:.1f}%)")


if __name__ == "__main__":
    main()
