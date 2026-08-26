"""
reproduce_table.py — Reproduce the main results table (Table 2, Section 4)
from the metrics.json files in data/results/.

Usage:
    uv run reproduce_table.py                          # default: data/results
    uv run reproduce_table.py --results-dir data/results
"""

import argparse
import json
import os
from pathlib import Path

LABELS = ["cascade_delete", "stale_aggregate", "exposed_record"]
LABEL_SHORT = {"cascade_delete": "CD", "stale_aggregate": "SA", "exposed_record": "ER"}
LABEL_DIM = {"cascade_delete": "retention", "stale_aggregate": "consistency", "exposed_record": "visibility"}

MODEL_DISPLAY = {
    "openai_gpt-5.4": "GPT-5.4",
    "together_ai_moonshotai_Kimi-K2.6": "Kimi-K2.6",
    "together_ai_deepseek-ai_DeepSeek-V3.1": "DeepSeek-V3.1",
    "google_gemma-4-31b-it": "Gemma-4-31b",
    "together_ai_Qwen_Qwen3.5-397B-A17B": "Qwen3.5-397B-A17B"
}

MODEL_ORDER = list(MODEL_DISPLAY.keys())


def load_metrics(path: str) -> dict:
    with open(path) as f:
        return json.load(f)


def find_runs(model_dir: str) -> dict:
    """Return {"free": {label: metrics}, "definition": {label: metrics}} for a model."""
    result = {"free": {}, "definition": {}}

    for prompt_type in ("free", "definition"):
        prompt_dir = os.path.join(model_dir, prompt_type)
        if not os.path.isdir(prompt_dir):
            continue

        timestamps = sorted(
            t for t in os.listdir(prompt_dir)
            if not t.endswith("_reasoning") and not t.startswith(".")
            and os.path.isdir(os.path.join(prompt_dir, t))
        )

        if prompt_type == "free":
            # Pick the earliest timestamp that has all 3 labels
            for ts in timestamps:
                metrics_path = os.path.join(prompt_dir, ts, "metrics.json")
                if not os.path.exists(metrics_path):
                    continue
                data = load_metrics(metrics_path)
                agg = data.get("aggregate", {})
                if all(label in agg for label in LABELS):
                    for label in LABELS:
                        result["free"][label] = agg[label]
                    break
        else:
            # Each timestamp covers one label; pick the earliest per label
            for ts in timestamps:
                metrics_path = os.path.join(prompt_dir, ts, "metrics.json")
                if not os.path.exists(metrics_path):
                    continue
                data = load_metrics(metrics_path)
                agg = data.get("aggregate", {})
                for label in LABELS:
                    if label in agg and label not in result["definition"]:
                        result["definition"][label] = agg[label]

    return result


def fmt(val: float) -> str:
    return f"{val:.2f}"


def delta_str(def_f1: float, free_f1: float) -> str:
    diff = def_f1 - free_f1
    if abs(diff) < 0.005:
        return "(.00)"
    sign = "+" if diff > 0 else ""
    return f"({sign}{diff:.2f})"


def print_table(results_dir: str) -> None:
    results_dir = str(Path(results_dir).resolve())

    models_data = {}
    for model_key in MODEL_ORDER:
        model_dir = os.path.join(results_dir, model_key)
        if not os.path.isdir(model_dir):
            continue
        models_data[model_key] = find_runs(model_dir)

    # Header
    header_labels = "".join(f"  {LABEL_SHORT[l]:>12s} ({LABEL_DIM[l]})" for l in LABELS)
    print(f"{'Model':<20s} {'Prompt':<6s}{header_labels}")
    sub_header = "".join("     P     R    F1  delta" for _ in LABELS)
    print(f"{'':20s} {'':6s}{sub_header}")
    print("-" * 116)

    # Accumulators for average
    avg = {pt: {l: {"p": [], "r": [], "f1": []} for l in LABELS} for pt in ("free", "definition")}

    for model_key in MODEL_ORDER:
        if model_key not in models_data:
            continue
        data = models_data[model_key]
        display_name = MODEL_DISPLAY[model_key]

        for prompt_type in ("free", "definition"):
            row = f"{display_name if prompt_type == 'free' else '':20s} {'FREE' if prompt_type == 'free' else 'DEF':6s}"

            for label in LABELS:
                m = data.get(prompt_type, {}).get(label)
                if m is None:
                    row += "     -     -     -      "
                    continue

                p, r, f1 = m["precision"], m["recall"], m["f1"]
                avg[prompt_type][label]["p"].append(p)
                avg[prompt_type][label]["r"].append(r)
                avg[prompt_type][label]["f1"].append(f1)

                if prompt_type == "definition":
                    free_m = data.get("free", {}).get(label)
                    free_f1 = free_m["f1"] if free_m else 0.0
                    delta = delta_str(f1, free_f1)
                    row += f"  {fmt(p):>5s} {fmt(r):>5s} {fmt(f1):>5s} {delta:>7s}"
                else:
                    row += f"  {fmt(p):>5s} {fmt(r):>5s} {fmt(f1):>5s}        "

            print(row)
        print()

    # Average row
    print("-" * 116)
    for prompt_type in ("free", "definition"):
        row = f"{'Average' if prompt_type == 'free' else '':20s} {'FREE' if prompt_type == 'free' else 'DEF':6s}"

        for label in LABELS:
            vals = avg[prompt_type][label]
            if not vals["f1"]:
                row += "     -     -     -      "
                continue

            p = sum(vals["p"]) / len(vals["p"])
            r = sum(vals["r"]) / len(vals["r"])
            f1 = sum(vals["f1"]) / len(vals["f1"])

            if prompt_type == "definition":
                free_vals = avg["free"][label]
                free_f1 = sum(free_vals["f1"]) / len(free_vals["f1"]) if free_vals["f1"] else 0.0
                delta = delta_str(f1, free_f1)
                row += f"  {fmt(p):>5s} {fmt(r):>5s} {fmt(f1):>5s} {delta:>7s}"
            else:
                row += f"  {fmt(p):>5s} {fmt(r):>5s} {fmt(f1):>5s}        "

        print(row)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Reproduce Table 2 from metrics.json files")
    parser.add_argument("--results-dir", default="data/results", help="Path to results directory")
    args = parser.parse_args()
    print_table(args.results_dir)
