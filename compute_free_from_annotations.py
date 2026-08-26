"""
compute_free_from_annotations.py — Recompute FREE results from manual annotations,
replacing the LLM judge entirely.

Reads the *_annotated.csv files from manual_review/ and computes P/R/F1
per model and issue category, matching the format of Table 2 in the paper.

Usage:
    uv run compute_free_from_annotations.py
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

LABELS = ["exposed_record", "stale_aggregate", "cascade_delete"]
LABEL_SHORT = {"cascade_delete": "CD", "stale_aggregate": "SA", "exposed_record": "ER"}


def compute_metrics(tp, fp, tn, fn):
    p = tp / (tp + fp) if (tp + fp) > 0 else 0.0
    r = tp / (tp + fn) if (tp + fn) > 0 else 0.0
    f1 = 2 * p * r / (p + r) if (p + r) > 0 else 0.0
    return p, r, f1


def load_annotations(filepath: Path) -> list[dict]:
    rows = []
    with open(filepath, newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f, delimiter=";")
        for row in reader:
            rows.append(row)
    return rows


def main():
    # Accumulators for average row
    avg_counts = {label: {"tp": 0, "fp": 0, "tn": 0, "fn": 0} for label in LABELS}
    n_models = 0

    # Also track per-model counts for the average
    per_model_metrics = {}

    # Also compare judge vs manual
    print("=" * 90)
    print("JUDGE vs MANUAL AGREEMENT")
    print("=" * 90)
    total_agree = 0
    total_total = 0
    for model_name, fname in ANNOTATED_FILES.items():
        rows = load_annotations(REVIEW_DIR / fname)
        agree = sum(1 for r in rows if r["judge_detected"].strip().upper() == r["my_classification"].strip().upper())
        print(f"  {model_name:20s}  {agree}/{len(rows)} ({agree/len(rows)*100:.1f}%)")
        total_agree += agree
        total_total += len(rows)
    print(f"  {'TOTAL':20s}  {total_agree}/{total_total} ({total_agree/total_total*100:.1f}%)")

    # ================================================================
    # MAIN TABLE: Per-model, per-label metrics from manual annotations
    # ================================================================
    print(f"\n\n{'='*90}")
    print("FREE RESULTS FROM MANUAL ANNOTATIONS")
    print("=" * 90)
    print(f"\n{'Model':20s} {'':5s}", end="")
    for label in LABELS:
        print(f"  {LABEL_SHORT[label]:>12s}", end="")
    print()
    print(f"{'':20s} {'':5s}", end="")
    for _ in LABELS:
        print(f"  {'P':>4s} {'R':>4s} {'F1':>5s}", end="")
    print()
    print("-" * 75)

    for model_name, fname in ANNOTATED_FILES.items():
        rows = load_annotations(REVIEW_DIR / fname)
        n_models += 1

        counts = {label: {"tp": 0, "fp": 0, "tn": 0, "fn": 0} for label in LABELS}

        for row in rows:
            label = row["label"].strip()
            gt = row["gt_detected"].strip().upper() == "TRUE"
            my = row["my_classification"].strip().upper() == "TRUE"

            if gt and my:
                counts[label]["tp"] += 1
            elif gt and not my:
                counts[label]["fn"] += 1
            elif not gt and my:
                counts[label]["fp"] += 1
            else:
                counts[label]["tn"] += 1

        line = f"{model_name:20s} FREE "
        model_metrics_row = {}
        for label in LABELS:
            c = counts[label]
            p, r, f1 = compute_metrics(c["tp"], c["fp"], c["tn"], c["fn"])
            line += f"  {p:4.2f} {r:4.2f} {f1:5.2f}"
            model_metrics_row[label] = (p, r, f1)
            for k in ["tp", "fp", "tn", "fn"]:
                avg_counts[label][k] += c[k]

        per_model_metrics[model_name] = model_metrics_row
        print(line)

    # Average
    print("-" * 75)
    line = f"{'Average':20s} FREE "
    for label in LABELS:
        c = avg_counts[label]
        p, r, f1 = compute_metrics(c["tp"], c["fp"], c["tn"], c["fn"])
        line += f"  {p:4.2f} {r:4.2f} {f1:5.2f}"
    print(line)

    # ================================================================
    # LaTeX-ready rows
    # ================================================================
    print(f"\n\n{'='*90}")
    print("LATEX TABLE ROWS (FREE only)")
    print("=" * 90)

    for model_name in ANNOTATED_FILES:
        m = per_model_metrics[model_name]
        er_p, er_r, er_f1 = m["exposed_record"]
        sa_p, sa_r, sa_f1 = m["stale_aggregate"]
        cd_p, cd_r, cd_f1 = m["cascade_delete"]
        print(f"  & \\textsc{{Free}} & {er_p:.2f} & {er_r:.2f} & \\colorcell{{{er_f1:.2f}}} &"
              f"          & {sa_p:.2f} & {sa_r:.2f} & \\colorcell{{{sa_f1:.2f}}} &"
              f"          & {cd_p:.2f} & {cd_r:.2f} & \\colorcell{{{cd_f1:.2f}}} & \\\\")

    # Average row
    print("  % Average:")
    er_p, er_r, er_f1 = compute_metrics(avg_counts["exposed_record"]["tp"], avg_counts["exposed_record"]["fp"],
                                         avg_counts["exposed_record"]["tn"], avg_counts["exposed_record"]["fn"])
    sa_p, sa_r, sa_f1 = compute_metrics(avg_counts["stale_aggregate"]["tp"], avg_counts["stale_aggregate"]["fp"],
                                         avg_counts["stale_aggregate"]["tn"], avg_counts["stale_aggregate"]["fn"])
    cd_p, cd_r, cd_f1 = compute_metrics(avg_counts["cascade_delete"]["tp"], avg_counts["cascade_delete"]["fp"],
                                         avg_counts["cascade_delete"]["tn"], avg_counts["cascade_delete"]["fn"])
    print(f"  & \\textsc{{Free}} & {er_p:.2f} & {er_r:.2f} & \\colorcell{{{er_f1:.2f}}} &"
          f"          & {sa_p:.2f} & {sa_r:.2f} & \\colorcell{{{sa_f1:.2f}}} &"
          f"          & {cd_p:.2f} & {cd_r:.2f} & \\colorcell{{{cd_f1:.2f}}} & \\\\")

    # ================================================================
    # Per-label breakdown with counts
    # ================================================================
    print(f"\n\n{'='*90}")
    print("DETAILED COUNTS PER MODEL PER LABEL")
    print("=" * 90)
    for model_name, fname in ANNOTATED_FILES.items():
        rows = load_annotations(REVIEW_DIR / fname)
        print(f"\n  {model_name}:")
        for label in LABELS:
            label_rows = [r for r in rows if r["label"].strip() == label]
            tp = sum(1 for r in label_rows if r["gt_detected"].strip().upper() == "TRUE" and r["my_classification"].strip().upper() == "TRUE")
            fp = sum(1 for r in label_rows if r["gt_detected"].strip().upper() == "FALSE" and r["my_classification"].strip().upper() == "TRUE")
            tn = sum(1 for r in label_rows if r["gt_detected"].strip().upper() == "FALSE" and r["my_classification"].strip().upper() == "FALSE")
            fn = sum(1 for r in label_rows if r["gt_detected"].strip().upper() == "TRUE" and r["my_classification"].strip().upper() == "FALSE")
            p, r, f1 = compute_metrics(tp, fp, tn, fn)
            print(f"    {LABEL_SHORT[label]:3s}: TP={tp:2d} FP={fp:2d} TN={tn:2d} FN={fn:2d}  P={p:.2f} R={r:.2f} F1={f1:.2f}")

    # ================================================================
    # Variant breakdown (for appendix tables)
    # ================================================================
    print(f"\n\n{'='*90}")
    print("VARIANT BREAKDOWN (all models, manual annotations)")
    print("=" * 90)

    def variant_to_lang(v):
        if v.startswith("python"): return "Python"
        if v.startswith("js"): return "JavaScript"
        if v.startswith("go"): return "Go"
        return "Unknown"

    def variant_to_access(v):
        parts = v.split("_")
        return "Raw SQL" if parts[2] == "rawsql" else "ORM"

    all_rows = []
    for fname in ANNOTATED_FILES.values():
        all_rows.extend(load_annotations(REVIEW_DIR / fname))

    # By language
    print("\n  By language (FREE, all models):")
    for lang in ["Python", "JavaScript", "Go"]:
        for label in LABELS:
            lr = [r for r in all_rows if variant_to_lang(r["variant"].strip()) == lang and r["label"].strip() == label]
            tp = sum(1 for r in lr if r["gt_detected"].strip().upper() == "TRUE" and r["my_classification"].strip().upper() == "TRUE")
            fp = sum(1 for r in lr if r["gt_detected"].strip().upper() == "FALSE" and r["my_classification"].strip().upper() == "TRUE")
            fn = sum(1 for r in lr if r["gt_detected"].strip().upper() == "TRUE" and r["my_classification"].strip().upper() == "FALSE")
            tn = sum(1 for r in lr if r["gt_detected"].strip().upper() == "FALSE" and r["my_classification"].strip().upper() == "FALSE")
            p, r_, f1 = compute_metrics(tp, fp, tn, fn)
            print(f"    {lang:12s} {LABEL_SHORT[label]:3s}  F1={f1:.2f}")

    # By access
    print("\n  By data-access (FREE, all models):")
    for access in ["Raw SQL", "ORM"]:
        for label in LABELS:
            lr = [r for r in all_rows if variant_to_access(r["variant"].strip()) == access and r["label"].strip() == label]
            tp = sum(1 for r in lr if r["gt_detected"].strip().upper() == "TRUE" and r["my_classification"].strip().upper() == "TRUE")
            fp = sum(1 for r in lr if r["gt_detected"].strip().upper() == "FALSE" and r["my_classification"].strip().upper() == "TRUE")
            fn = sum(1 for r in lr if r["gt_detected"].strip().upper() == "TRUE" and r["my_classification"].strip().upper() == "FALSE")
            tn = sum(1 for r in lr if r["gt_detected"].strip().upper() == "FALSE" and r["my_classification"].strip().upper() == "FALSE")
            p, r_, f1 = compute_metrics(tp, fp, tn, fn)
            print(f"    {access:12s} {LABEL_SHORT[label]:3s}  F1={f1:.2f}")


if __name__ == "__main__":
    main()
