"""
compare_neutral_ablation.py — Reproduce the neutrality ablation table
(paper Appendix E.1, Table 7) from pre-computed results.

Compares the original DEF run on cascade_delete against the run whose prompt
adds a neutrality cue explicitly licensing a negative answer
(see run_ablation_neutral.py), for GPT-5.4 and Kimi-K2.6.

Usage:
    uv run compare_neutral_ablation.py
"""

import json
from pathlib import Path

LABEL = "cascade_delete"

# Original DEF run directories — same selection as reproduce_table.py
# (earliest non-reasoning timestamp per label)
ORIGINAL = {
    "GPT-5.4": Path("data/results/openai_gpt-5.4/definition/20260513_011923"),
    "Kimi-K2.6": Path("data/results/together_ai_moonshotai_Kimi-K2.6/definition/20260520_164134"),
}

# Neutrality-cue run directories
NEUTRAL = {
    "GPT-5.4": Path("data/results_ablation_neutral/openai_gpt-5.4/definition/20260708_200755"),
    "Kimi-K2.6": Path("data/results_ablation_neutral/together_ai_moonshotai_Kimi-K2.6/definition/20260708_201222"),
}


def load(run_dir: Path) -> dict:
    m = json.loads((run_dir / "metrics.json").read_text())
    agg = m["aggregate"][LABEL]
    return {
        "p": agg["precision"], "r": agg["recall"], "f1": agg["f1"],
        "tn": int(agg["tn"]), "fp": int(agg["fp"]),
    }


def main():
    print("Neutrality ablation on cascade_delete (DEF, 36 positive + 36 negative instances)")
    print(f"{'Model':<12s}  {'Prompt':<13s}  {'P':>5s} {'R':>5s} {'F1':>6s}  {'TN':>6s}")
    print("-" * 56)
    for model in ORIGINAL:
        for tag, runs in (("original", ORIGINAL), ("+ neutrality", NEUTRAL)):
            a = load(runs[model])
            print(f"{model:<12s}  {tag:<13s}  {a['p']:5.2f} {a['r']:5.2f} {a['f1']:6.2f}  {a['tn']:3d}/36")
        print()
    print("If the neutrality cue left over-flagging intact, TN barely moves: the")
    print("compliance bias is not explained by instruction-following pressure.")


if __name__ == "__main__":
    main()
