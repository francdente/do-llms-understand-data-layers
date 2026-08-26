"""
compare_base_augmented.py — Hand-crafted base tasks vs. GPT-5.4 augmentations.

Splits every (model, category, regime) result into the 18 hand-written base
instances (variant python_flask_rawsql) and the 198 LLM-generated variants,
and reports:

  1. Pooled P/R/F1 per category and regime, base vs augmented (5 models).
  2. Per-model F1 on each split, and the difference-in-differences
     DiD = (F1_aug - F1_base) for GPT-5.4 minus the mean of the other four
     models. If augmentation favoured the generator, DiD should be positive
     and large.
  3. Two-sided Fisher exact tests on the underlying correct/incorrect counts
     (base vs augmented) per model, category and regime.

Data sources are identical to bootstrap_ci.py / the paper's Table 2:
  FREE: manual_review/*_annotated.csv   DEF: data/results/.../metrics.json

Usage:
    uv run compare_base_augmented.py [--markdown]
"""

import argparse
import csv
import json
import os
import re
from collections import defaultdict
from math import comb
from pathlib import Path

from bootstrap_ci import (
    MODELS,
    LABELS,
    LABEL_SHORT,
    RESULTS_DIR,
    REVIEW_DIR,
    f1_from_counts,
)

BASE_VARIANT = "python_flask_rawsql"

DIM = {"cascade_delete": "retention",
       "stale_aggregate": "consistency",
       "exposed_record": "visibility"}


def load_free_by_variant(csv_name: str):
    """-> {label: {variant: [(tp,fp,tn,fn), ...]}}"""
    data = defaultdict(lambda: defaultdict(list))
    with open(REVIEW_DIR / csv_name, newline="", encoding="utf-8") as f:
        for row in csv.DictReader(f, delimiter=";"):
            gt = row["gt_detected"].strip().upper() == "TRUE"
            pred = row["my_classification"].strip().upper() == "TRUE"
            data[row["label"].strip()][row["variant"].strip()].append(
                (int(gt and pred), int(not gt and pred),
                 int(not gt and not pred), int(gt and not pred)))
    return data


def load_def_by_variant(model_dir_name: str):
    """-> {label: {variant: [(tp,fp,tn,fn), ...]}} (same run selection as bootstrap_ci)"""
    data = defaultdict(lambda: defaultdict(list))
    prompt_dir = RESULTS_DIR / model_dir_name / "definition"
    timestamps = sorted(
        t for t in os.listdir(prompt_dir)
        if not t.endswith("_reasoning") and not t.startswith(".")
        and (prompt_dir / t).is_dir()
    )
    for ts in timestamps:
        mpath = prompt_dir / ts / "metrics.json"
        if not mpath.exists():
            continue
        m = json.loads(mpath.read_text())
        for label in LABELS:
            if label in m.get("aggregate", {}) and label not in data:
                for d in m["runs"][0]["details"]:
                    if d["label"] == label:
                        data[label][d["variant"]].append(
                            (d["tp"], d["fp"], d["tn"], d["fn"]))
    return data


def counts(by_variant: dict, base: bool):
    """Pool (tp, fp, tn, fn) over the base variant or over the 11 others."""
    tp = fp = tn = fn = 0
    for variant, rows in by_variant.items():
        if (variant == BASE_VARIANT) != base:
            continue
        for (i_tp, i_fp, i_tn, i_fn) in rows:
            tp += i_tp
            fp += i_fp
            tn += i_tn
            fn += i_fn
    return tp, fp, tn, fn


def prf(tp, fp, fn):
    p = tp / (tp + fp) if (tp + fp) else 0.0
    r = tp / (tp + fn) if (tp + fn) else 0.0
    return p, r, f1_from_counts(tp, fp, fn)


def fisher_exact_two_sided(a, b, c, d):
    """2x2 table [[a,b],[c,d]] -> two-sided p (sum of tables at most as likely)."""
    n = a + b + c + d
    r1, r2, c1 = a + b, c + d, a + c

    def prob(x):
        return (comb(r1, x) * comb(r2, c1 - x)) / comb(n, c1)

    p_obs = prob(a)
    lo, hi = max(0, c1 - r2), min(r1, c1)
    return min(1.0, sum(prob(x) for x in range(lo, hi + 1)
                        if prob(x) <= p_obs * (1 + 1e-9)))


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--markdown", action="store_true")
    args = ap.parse_args()

    free = {name: load_free_by_variant(csv_name)
            for name, (_, csv_name) in MODELS.items()}
    dfn = {name: load_def_by_variant(dir_name)
           for name, (dir_name, _) in MODELS.items()}
    regimes = {"FREE": free, "DEF": dfn}

    # ---- 1. Pooled base vs augmented -------------------------------------
    print("=" * 88)
    print("MACRO-AVERAGED OVER THE 5 MODELS: BASE (n=18) vs AUGMENTED (n=198)")
    print("(macro-average of per-model P/R/F1, matching the Average row of Table 2)")
    print("=" * 88)
    if args.markdown:
        print("| Category | Regime | Base P/R/F1 | Augmented P/R/F1 |")
        print("|---|---|---|---|")
    else:
        print(f"{'Category':32s} {'Regime':6s}   {'Base P/R/F1':>20s}   {'Augmented P/R/F1':>20s}")

    for label in ("cascade_delete", "stale_aggregate", "exposed_record"):
        for regime, data in regimes.items():
            macro = {}
            for is_base in (True, False):
                rows = []
                for name in MODELS:
                    tp, fp, tn, fn = counts(data[name][label], is_base)
                    rows.append(prf(tp, fp, fn))
                macro[is_base] = tuple(sum(v[i] for v in rows) / len(rows)
                                       for i in range(3))
            bp, br, bf = macro[True]
            ap_, ar, af = macro[False]
            name_ = f"{label} ({DIM[label]})"
            if args.markdown:
                print(f"| {name_} | {regime} | "
                      f"{bp:.2f}/{br:.2f}/{bf:.2f} | {ap_:.2f}/{ar:.2f}/{af:.2f} |")
            else:
                print(f"{name_:32s} {regime:6s}   "
                      f"{bp:6.2f}/{br:.2f}/{bf:.2f}   {ap_:14.2f}/{ar:.2f}/{af:.2f}")

    # ---- 2. Difference-in-differences ------------------------------------
    print()
    print("=" * 88)
    print("DIFFERENCE-IN-DIFFERENCES (Delta = F1_augmented - F1_base)")
    print("=" * 88)
    if args.markdown:
        print("| Category | Regime | D GPT-5.4 | mean D others | DiD |")
        print("|---|---|---|---|---|")
    else:
        print(f"{'Category':32s} {'Regime':6s} {'D GPT-5.4':>10s} {'mean D others':>14s} {'DiD':>7s}")

    per_model_f1 = {}  # (model, label, regime, is_base) -> f1
    for name in MODELS:
        for label in LABELS:
            for regime, data in regimes.items():
                for is_base in (True, False):
                    tp, fp, tn, fn = counts(data[name][label], is_base)
                    per_model_f1[(name, label, regime, is_base)] = f1_from_counts(tp, fp, fn)

    for label in ("cascade_delete", "stale_aggregate", "exposed_record"):
        for regime in regimes:
            def delta(m):
                return (per_model_f1[(m, label, regime, False)]
                        - per_model_f1[(m, label, regime, True)])
            d_gpt = delta("GPT-5.4")
            others = [delta(m) for m in MODELS if m != "GPT-5.4"]
            d_oth = sum(others) / len(others)
            name_ = f"{label} ({DIM[label]})"
            if args.markdown:
                print(f"| {name_} | {regime} | {d_gpt:+.2f} | {d_oth:+.2f} | {d_gpt - d_oth:+.2f} |")
            else:
                print(f"{name_:32s} {regime:6s} {d_gpt:+10.2f} {d_oth:+14.2f} {d_gpt - d_oth:+7.2f}")

    # ---- 3. Fisher exact on correct/incorrect counts ----------------------
    print()
    print("=" * 88)
    print("FISHER EXACT (correct vs incorrect instances, base vs augmented), per model")
    print("=" * 88)
    print(f"{'Model':20s} {'Cat':4s} {'Regime':6s} {'base ok/n':>10s} {'aug ok/n':>10s} {'p':>8s}")
    min_p = (1.0, None)
    for name in MODELS:
        for label in LABELS:
            for regime, data in regimes.items():
                tp, fp, tn, fn = counts(data[name][label], True)
                b_ok, b_bad = tp + tn, fp + fn
                tp, fp, tn, fn = counts(data[name][label], False)
                a_ok, a_bad = tp + tn, fp + fn
                p = fisher_exact_two_sided(b_ok, b_bad, a_ok, a_bad)
                if p < min_p[0]:
                    min_p = (p, (name, label, regime))
                print(f"{name:20s} {LABEL_SHORT[label]:4s} {regime:6s} "
                      f"{b_ok:4d}/{b_ok + b_bad:<5d} {a_ok:4d}/{a_ok + a_bad:<5d} {p:8.3f}")
    print(f"\nSmallest p across all 30 tests: {min_p[0]:.3f} at {min_p[1]}"
          f"  (Bonferroni threshold 0.05/30 = {0.05 / 30:.4f})")


if __name__ == "__main__":
    main()
