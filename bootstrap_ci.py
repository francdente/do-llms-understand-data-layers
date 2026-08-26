"""
bootstrap_ci.py — Cluster bootstrap confidence intervals for the paper's
results (Appendix K, Table 16).

Instances derived from the same base task are correlated (12 variants each),
so CIs are computed by resampling the *base tasks* (the independent units),
stratified by label polarity (3 positive, 3 negative per category), keeping
all variants of a sampled task together.

Data sources (identical to the paper's scoring):
  FREE: manual_review/*_annotated.csv  (gt_detected vs my_classification)
  DEF:  data/results/<model>/definition/<ts>/metrics.json (runs[0].details),
        earliest timestamp per label, excluding *_reasoning runs
        (same selection as reproduce_table.py)

For each (model, category) we report 95% percentile CIs for FREE F1, DEF F1,
and the paired difference DEF-FREE (same task resample applied to both
regimes). The average row resamples tasks once per iteration and macro-
averages the per-model F1s, matching Table 2's average row.

Usage:
    uv run bootstrap_ci.py [--iters 10000] [--seed 0]
"""

import argparse
import csv
import json
import os
import random
import re
from collections import defaultdict
from pathlib import Path

csv.field_size_limit(10_000_000)

RESULTS_DIR = Path("data/results")
REVIEW_DIR = Path("manual_review")

LABELS = ["exposed_record", "stale_aggregate", "cascade_delete"]
LABEL_SHORT = {"cascade_delete": "CD", "stale_aggregate": "SA", "exposed_record": "ER"}

MODELS = {
    "GPT-5.4": ("openai_gpt-5.4", "openai_gpt-5.4_annotated.csv"),
    "Kimi-K2.6": ("together_ai_moonshotai_Kimi-K2.6", "together_ai_moonshotai_Kimi-K2.6_annotated.csv"),
    "DeepSeek-V3.1": ("together_ai_deepseek-ai_DeepSeek-V3.1", "together_ai_deepseek-ai_DeepSeek-V3.1_annotated.csv"),
    "Gemma-4-31b": ("google_gemma-4-31b-it", "google_gemma-4-31b-it_annotated.csv"),
    "Qwen3.5-397B-A17B": ("together_ai_Qwen_Qwen3.5-397B-A17B", "together_ai_Qwen_Qwen3.5-397B-A17B_annotated.csv"),
}


def is_positive_task(task: str) -> bool:
    m = re.search(r"_([PN])\d", task)
    if not m:
        raise ValueError(f"Cannot parse polarity from task name: {task}")
    return m.group(1) == "P"


def f1_from_counts(tp, fp, fn):
    p = tp / (tp + fp) if (tp + fp) > 0 else 0.0
    r = tp / (tp + fn) if (tp + fn) > 0 else 0.0
    return 2 * p * r / (p + r) if (p + r) > 0 else 0.0


def load_free(csv_name: str):
    """-> {label: {task: [(tp,fp,tn,fn), ...]}}"""
    data = defaultdict(lambda: defaultdict(list))
    with open(REVIEW_DIR / csv_name, newline="", encoding="utf-8") as f:
        for row in csv.DictReader(f, delimiter=";"):
            label = row["label"].strip()
            gt = row["gt_detected"].strip().upper() == "TRUE"
            pred = row["my_classification"].strip().upper() == "TRUE"
            counts = (int(gt and pred), int(not gt and pred),
                      int(not gt and not pred), int(gt and not pred))
            # (tp, fp, tn, fn)
            data[label][row["task"].strip()].append(counts)
    return data


def load_def(model_dir_name: str):
    """Earliest non-reasoning timestamp per label, runs[0].details.
    -> {label: {task: [(tp,fp,tn,fn), ...]}}"""
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
        agg = m.get("aggregate", {})
        for label in LABELS:
            if label in agg and label not in data:
                for d in m["runs"][0]["details"]:
                    if d["label"] == label:
                        data[label][d["task"]].append(
                            (d["tp"], d["fp"], d["tn"], d["fn"]))
    return data


def f1_of_tasks(task_counts: dict, tasks: list) -> float:
    tp = fp = fn = 0
    for t in tasks:
        for (i_tp, i_fp, _i_tn, i_fn) in task_counts.get(t, []):
            tp += i_tp
            fp += i_fp
            fn += i_fn
    return f1_from_counts(tp, fp, fn)


def pct_ci(values, lo=2.5, hi=97.5):
    s = sorted(values)
    def pct(q):
        idx = q / 100 * (len(s) - 1)
        f, c = int(idx), min(int(idx) + 1, len(s) - 1)
        return s[f] + (s[c] - s[f]) * (idx - f)
    return pct(lo), pct(hi)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--iters", type=int, default=10_000)
    ap.add_argument("--seed", type=int, default=0)
    ap.add_argument("--latex", action="store_true", help="Also print LaTeX table rows")
    args = ap.parse_args()
    rng = random.Random(args.seed)

    free_data = {name: load_free(csv_name) for name, (_, csv_name) in MODELS.items()}
    def_data = {name: load_def(dir_name) for name, (dir_name, _) in MODELS.items()}

    # Canonical task lists per label (union over regimes/models; sanity-checked)
    tasks_by_label = {}
    for label in LABELS:
        tasks = set()
        for name in MODELS:
            tasks |= set(free_data[name][label]) | set(def_data[name][label])
        pos = sorted(t for t in tasks if is_positive_task(t))
        neg = sorted(t for t in tasks if not is_positive_task(t))
        assert len(pos) == 3 and len(neg) == 3, (label, pos, neg)
        tasks_by_label[label] = (pos, neg)

    # Point estimates + bootstrap
    results = {}  # (model, label) -> dict
    boot_f1s = {}  # (model, label, regime) -> list of B floats (for average row)
    boot_deltas = {}

    for label in LABELS:
        pos, neg = tasks_by_label[label]
        # Pre-draw the task resamples once per label so that all models (and
        # the average row) share the same resamples -> paired comparisons.
        samples = [
            (rng.choices(pos, k=3), rng.choices(neg, k=3))
            for _ in range(args.iters)
        ]
        for name in MODELS:
            fd, dd = free_data[name][label], def_data[name][label]
            free_pt = f1_of_tasks(fd, pos + neg)
            def_pt = f1_of_tasks(dd, pos + neg)
            fs, ds, deltas = [], [], []
            for sp, sn in samples:
                tasks = sp + sn
                f = f1_of_tasks(fd, tasks)
                d = f1_of_tasks(dd, tasks)
                fs.append(f)
                ds.append(d)
                deltas.append(d - f)
            results[(name, label)] = {
                "free": free_pt, "free_ci": pct_ci(fs),
                "def": def_pt, "def_ci": pct_ci(ds),
                "delta": def_pt - free_pt, "delta_ci": pct_ci(deltas),
            }
            boot_f1s[(name, label, "free")] = fs
            boot_f1s[(name, label, "def")] = ds
            boot_deltas[(name, label)] = deltas

        # Average row: macro-average per iteration over the shared resamples
        fs = [sum(boot_f1s[(n, label, "free")][b] for n in MODELS) / len(MODELS)
              for b in range(args.iters)]
        ds = [sum(boot_f1s[(n, label, "def")][b] for n in MODELS) / len(MODELS)
              for b in range(args.iters)]
        deltas = [d - f for d, f in zip(ds, fs)]
        pos_neg = tasks_by_label[label][0] + tasks_by_label[label][1]
        free_pt = sum(f1_of_tasks(free_data[n][label], pos_neg) for n in MODELS) / len(MODELS)
        def_pt = sum(f1_of_tasks(def_data[n][label], pos_neg) for n in MODELS) / len(MODELS)
        results[("Average", label)] = {
            "free": free_pt, "free_ci": pct_ci(fs),
            "def": def_pt, "def_ci": pct_ci(ds),
            "delta": def_pt - free_pt, "delta_ci": pct_ci(deltas),
        }

    # Report
    def ci(t):
        return f"[{t[0]:.2f},{t[1]:.2f}]"

    print(f"Cluster bootstrap over base tasks (stratified 3 pos + 3 neg, "
          f"B={args.iters}, seed={args.seed}), 95% percentile CIs\n")
    header = (f"{'Model':<19s}{'Cat':<5s}"
              f"{'FREE F1':>8s} {'95% CI':>12s}   "
              f"{'DEF F1':>7s} {'95% CI':>12s}   "
              f"{'Δ(D-F)':>7s} {'95% CI':>13s}  sig")
    print(header)
    print("-" * len(header))
    for name in list(MODELS) + ["Average"]:
        for label in ["exposed_record", "stale_aggregate", "cascade_delete"]:
            r = results[(name, label)]
            lo, hi = r["delta_ci"]
            sig = "*" if (lo > 0 or hi < 0) else " "
            print(f"{name:<19s}{LABEL_SHORT[label]:<5s}"
                  f"{r['free']:>8.2f} {ci(r['free_ci']):>12s}   "
                  f"{r['def']:>7.2f} {ci(r['def_ci']):>12s}   "
                  f"{r['delta']:>+7.2f} {ci(r['delta_ci']):>13s}  {sig}")
        print()

    print("* = 95% CI of the paired DEF-FREE difference excludes 0")

    if args.latex:
        def lci(t):
            return f"[{t[0]:.2f}, {t[1]:.2f}]"

        print("\n% LaTeX rows (Model & Cat & FREE F1 CI & DEF F1 CI & Delta CI)")
        for name in list(MODELS) + ["Average"]:
            display = f"\\textit{{{name}}}" if name == "Average" else name
            if name == "Average":
                print("\\midrule")
            for i, label in enumerate(["exposed_record", "stale_aggregate", "cascade_delete"]):
                r = results[(name, label)]
                lo, hi = r["delta_ci"]
                sig = "$^{*}$" if (lo > 0 or hi < 0) else ""
                model_cell = f"\\multirow{{3}}{{*}}{{{display}}}" if i == 0 else ""
                print(f"{model_cell} & {LABEL_SHORT[label]}"
                      f" & {r['free']:.2f} {lci(r['free_ci'])}"
                      f" & {r['def']:.2f} {lci(r['def_ci'])}"
                      f" & {r['delta']:+.2f} {lci(r['delta_ci'])}{sig} \\\\")
            if name != "Average":
                print("\\midrule")


if __name__ == "__main__":
    main()
