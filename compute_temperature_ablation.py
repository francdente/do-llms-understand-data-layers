"""
compute_temperature_ablation.py — Reproduce the temperature-sensitivity table
(paper Appendix G, Table 11) from pre-computed results.

Compares the main temperature-0 runs against 3 independent runs at
temperature 0.7 for GPT-5.4 (DEF and FREE) and Qwen3.5-397B (DEF), reporting
the mean F1 across the T=0.7 runs and the population standard deviation.

Note: FREE rows use the LLM judge's classification for both temperatures
(re-annotating 3 extra runs manually was not practical), which is why the
GPT-5.4 FREE T=0 values here can differ by a point from Table 2, which uses
the manual annotations.

Usage:
    uv run compute_temperature_ablation.py
"""

import json
import statistics
from pathlib import Path

LABELS = ["cascade_delete", "stale_aggregate", "exposed_record"]
LABEL_SHORT = {"cascade_delete": "CD", "stale_aggregate": "SA", "exposed_record": "ER"}

# Main temperature-0 runs (same selection as reproduce_table.py)
T0 = {
    ("GPT-5.4", "DEF"): {
        "cascade_delete": Path("data/results/openai_gpt-5.4/definition/20260513_011923"),
        "stale_aggregate": Path("data/results/openai_gpt-5.4/definition/20260513_012219"),
        "exposed_record": Path("data/results/openai_gpt-5.4/definition/20260513_012254"),
    },
    ("GPT-5.4", "FREE"): {
        label: Path("data/results/openai_gpt-5.4/free/20260512_180041")
        for label in LABELS
    },
    ("Qwen3.5-397B", "DEF"): {
        "cascade_delete": Path("data/results/together_ai_Qwen_Qwen3.5-397B-A17B/definition/20260521_164353"),
        "stale_aggregate": Path("data/results/together_ai_Qwen_Qwen3.5-397B-A17B/definition/20260521_165327"),
        "exposed_record": Path("data/results/together_ai_Qwen_Qwen3.5-397B-A17B/definition/20260521_165352"),
    },
}

# Temperature-0.7 runs (3 independent runs each)
T07 = {
    ("GPT-5.4", "DEF"): {
        "cascade_delete": Path("data/results/openai_gpt-5.4/definition/20260526_104359"),
        "stale_aggregate": Path("data/results/openai_gpt-5.4/definition/20260526_104501"),
        "exposed_record": Path("data/results/openai_gpt-5.4/definition/20260526_104743"),
    },
    ("GPT-5.4", "FREE"): {
        label: Path("data/results/openai_gpt-5.4/free/20260526_113410")
        for label in LABELS
    },
    ("Qwen3.5-397B", "DEF"): {
        "cascade_delete": Path("data/results/together_ai_Qwen_Qwen3.5-397B-A17B/definition/20260526_112755"),
        "stale_aggregate": Path("data/results/together_ai_Qwen_Qwen3.5-397B-A17B/definition/20260526_112932"),
        "exposed_record": Path("data/results/together_ai_Qwen_Qwen3.5-397B-A17B/definition/20260526_113203"),
    },
}


def run_f1s(run_dir: Path, label: str) -> list[float]:
    """Per-run F1 for `label` (DEF runs store one label; FREE runs store all three)."""
    m = json.loads((run_dir / "metrics.json").read_text())
    out = []
    for r in m["runs"]:
        by_label = r.get("by_label")
        if by_label is not None:  # FREE metrics carry a per-label breakdown
            out.append(by_label[label]["f1"])
        else:  # DEF metrics: recompute from the run's details
            tp = sum(d["tp"] for d in r["details"] if d["label"] == label)
            fp = sum(d["fp"] for d in r["details"] if d["label"] == label)
            fn = sum(d["fn"] for d in r["details"] if d["label"] == label)
            p = tp / (tp + fp) if tp + fp else 0.0
            rc = tp / (tp + fn) if tp + fn else 0.0
            out.append(2 * p * rc / (p + rc) if p + rc else 0.0)
    return out


def main():
    print("Temperature sensitivity: T=0 (main runs) vs T=0.7 (3 runs, mean and std dev)")
    print(f"{'Model':<14s} {'Prompt':<6s} {'Issue':<5s} {'T=0':>6s} {'T=0.7':>7s} {'sigma':>6s}")
    print("-" * 50)
    for (model, prompt) in T0:
        for label in LABELS:
            t0 = run_f1s(T0[(model, prompt)][label], label)[0]
            f1s = run_f1s(T07[(model, prompt)][label], label)
            mean = statistics.mean(f1s)
            sigma = statistics.pstdev(f1s)
            print(f"{model:<14s} {prompt:<6s} {LABEL_SHORT[label]:<5s} "
                  f"{t0:6.2f} {mean:7.2f} {sigma:6.2f}")
        print()


if __name__ == "__main__":
    main()
