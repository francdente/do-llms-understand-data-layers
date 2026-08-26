"""
compare_paraphrase_ablation.py — Compare original (P1) vs paraphrased (P2) DEF results.

Usage:
    uv run compare_paraphrase_ablation.py
"""

import json
from pathlib import Path

LABELS = ["cascade_delete", "stale_aggregate", "exposed_record"]
LABEL_SHORT = {"cascade_delete": "CD", "stale_aggregate": "SA", "exposed_record": "ER"}

# Original (P1) run directories — non-reasoning, earliest timestamp per label
P1 = {
    "GPT-5.4": {
        "cascade_delete": Path("data/results/openai_gpt-5.4/definition/20260513_011923"),
        "stale_aggregate": Path("data/results/openai_gpt-5.4/definition/20260513_012219"),
        "exposed_record": Path("data/results/openai_gpt-5.4/definition/20260513_012254"),
    },
    "Kimi-K2.6": {
        "cascade_delete": Path("data/results/together_ai_moonshotai_Kimi-K2.6/definition/20260520_164134"),
        "stale_aggregate": Path("data/results/together_ai_moonshotai_Kimi-K2.6/definition/20260513_154823"),
        "exposed_record": Path("data/results/together_ai_moonshotai_Kimi-K2.6/definition/20260513_155139"),
    },
}

# Paraphrased (P2) run directories
P2 = {
    "GPT-5.4": {
        "cascade_delete": Path("data/results_ablation_paraphrase/openai_gpt-5.4/definition/20260522_150857"),
        "stale_aggregate": Path("data/results_ablation_paraphrase/openai_gpt-5.4/definition/20260522_150903"),
        "exposed_record": Path("data/results_ablation_paraphrase/openai_gpt-5.4/definition/20260522_150913"),
    },
    "Kimi-K2.6": {
        "cascade_delete": Path("data/results_ablation_paraphrase/together_ai_moonshotai_Kimi-K2.6/definition/20260522_150922"),
        "stale_aggregate": Path("data/results_ablation_paraphrase/together_ai_moonshotai_Kimi-K2.6/definition/20260522_150937"),
        "exposed_record": Path("data/results_ablation_paraphrase/together_ai_moonshotai_Kimi-K2.6/definition/20260522_151942"),
    },
}


def load_f1(run_dir: Path, label: str) -> dict:
    m = json.loads((run_dir / "metrics.json").read_text())
    agg = m["aggregate"][label]
    return {"p": agg["precision"], "r": agg["recall"], "f1": agg["f1"]}


def main():
    models = list(P1.keys())

    print(f"{'Model':<12s}  {'Label':<6s}  {'P1 P':>5s} {'P1 R':>5s} {'P1 F1':>6s}  {'P2 P':>5s} {'P2 R':>5s} {'P2 F1':>6s}  {'|ΔF1|':>6s}")
    print("-" * 72)

    all_deltas = []

    for model in models:
        for label in LABELS:
            p1 = load_f1(P1[model][label], label)
            p2 = load_f1(P2[model][label], label)
            delta = abs(p2["f1"] - p1["f1"])
            all_deltas.append(delta)

            print(
                f"{model:<12s}  {LABEL_SHORT[label]:<6s}"
                f"  {p1['p']:5.2f} {p1['r']:5.2f} {p1['f1']:6.2f}"
                f"  {p2['p']:5.2f} {p2['r']:5.2f} {p2['f1']:6.2f}"
                f"  {delta:6.2f}"
            )
        print()

    mean_delta = sum(all_deltas) / len(all_deltas)
    max_delta = max(all_deltas)
    print(f"Mean |ΔF1|: {mean_delta:.3f}")
    print(f"Max  |ΔF1|: {max_delta:.3f}")


if __name__ == "__main__":
    main()
