"""
per_task_table.py — Per-base-task results (correct outcomes across the 12 variants).

For each of the 18 base tasks, reports how many of the task's 12
language/framework/ORM variants were handled correctly, per model and
averaged, for both FREE and DEF. For positive tasks (Pi) correct means the
target issue was detected (tp); for negative tasks (Ni) correct means the
variant was not flagged (tn).

Data sources are identical to bootstrap_ci.py / the paper's Table 2.

Usage:
    uv run per_task_table.py [--markdown]
"""

import argparse

from bootstrap_ci import MODELS, LABELS, LABEL_SHORT, load_free, load_def, is_positive_task


N_VARIANTS = 12  # variants per base task; missing/unparseable responses count as errors


def correct_stats(task_counts: dict, task: str):
    """-> (correct, total) across a task's variants; correct = tp (positive
    tasks) or tn (negative tasks), i.e. tp+tn per instance."""
    rows = task_counts.get(task, [])
    correct = sum(tp + tn for (tp, _fp, tn, _fn) in rows)
    return correct, N_VARIANTS


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--markdown", action="store_true", help="Emit a compact markdown table (models averaged)")
    args = ap.parse_args()

    free_data = {name: load_free(csv_name) for name, (_, csv_name) in MODELS.items()}
    def_data = {name: load_def(dir_name) for name, (dir_name, _) in MODELS.items()}

    tasks_by_label = {}
    for label in LABELS:
        tasks = set()
        for name in MODELS:
            tasks |= set(free_data[name][label]) | set(def_data[name][label])
        tasks_by_label[label] = sorted(tasks, key=lambda t: (not is_positive_task(t), t))

    order = ["exposed_record", "stale_aggregate", "cascade_delete"]

    if args.markdown:
        print("| Category | Base task | Polarity | FREE correct | DEF correct |")
        print("|---|---|---|---|---|")
        for label in order:
            for task in tasks_by_label[label]:
                pol = "positive" if is_positive_task(task) else "negative"
                ff = ft = df = dt = 0
                for name in MODELS:
                    f, t = correct_stats(free_data[name][label], task)
                    ff, ft = ff + f, ft + t
                    d, t2 = correct_stats(def_data[name][label], task)
                    df, dt = df + d, dt + t2
                print(f"| {LABEL_SHORT[label]} | {task.replace('_test','')} | {pol} "
                      f"| {ff}/{ft} ({ff/ft:.0%}) | {df}/{dt} ({df/dt:.0%}) |")
        return

    # Full per-model table (for the appendix)
    model_names = list(MODELS)
    header = f"{'Task':<16s}{'Pol':<5s}" + "".join(f"{n[:12]:>14s}" for n in model_names) + f"{'Avg':>8s}"
    for label in order:
        print(f"\n=== {label} ===  (correct variants out of 12, FREE | DEF)")
        print(header)
        for task in tasks_by_label[label]:
            pol = "pos" if is_positive_task(task) else "neg"
            row = f"{task.replace('_test',''):<16s}{pol:<5s}"
            tot_f = tot_d = 0
            for name in model_names:
                f, tf = correct_stats(free_data[name][label], task)
                d, td = correct_stats(def_data[name][label], task)
                tot_f += f
                tot_d += d
                row += f"{f:>5d}|{d:<2d}      "[:14].rjust(14)
            row += f"{tot_f/len(model_names):>4.1f}|{tot_d/len(model_names):<4.1f}"
            print(row)


if __name__ == "__main__":
    main()
