"""
free_review_stats.py — How much do models actually report in open-ended review?

Supports the search-space argument in the discovery-gap analysis: if FREE
performance were limited by the size of the candidate space, models should
bury the target issue among many other findings. Instead, reviews are short,
and the strongest FREE model most often reports nothing at all on positive
instances.

For each model we parse the FREE review (a JSON array of findings) stored in
manual_review/*_annotated.csv and report the number of findings per review.

Usage:
    uv run free_review_stats.py
"""

import csv
import json
import re

from bootstrap_ci import MODELS, is_positive_task

csv.field_size_limit(10_000_000)

FENCE = re.compile(r"```(?:json)?\s*(.*?)```", re.S)


def parse_findings(raw: str):
    """-> number of reported findings, or None if the output is not parseable."""
    text = raw.strip()
    if not text:
        return None
    candidates = [text]
    m = FENCE.search(text)
    if m:
        candidates.insert(0, m.group(1).strip())
    # last resort: the outermost [...] block
    i, j = text.find("["), text.rfind("]")
    if i != -1 and j > i:
        candidates.append(text[i:j + 1])
    for c in candidates:
        try:
            v = json.loads(c)
        except Exception:
            continue
        if isinstance(v, list):
            return len(v)
        if isinstance(v, dict):  # single finding object
            return 1
    return None


def main():
    print(f"{'Model':18s} {'parsed':>7s} {'findings/review':>16s} {'max':>4s} "
          f"{'empty':>7s} {'empty (pos.)':>13s}")
    all_n = []
    for name, (_, csv_name) in MODELS.items():
        rows = list(csv.DictReader(
            open("manual_review/" + csv_name, newline="", encoding="utf-8"),
            delimiter=";"))
        ns, ns_pos, bad = [], [], 0
        for r in rows:
            n = parse_findings(r["raw_output"])
            if n is None:
                bad += 1
                continue
            ns.append(n)
            if is_positive_task(r["task"]):
                ns_pos.append(n)
        all_n += ns
        print(f"{name:18s} {len(ns):3d}/{len(rows):<3d} {sum(ns) / len(ns):16.2f} "
              f"{max(ns):4d} {sum(1 for x in ns if x == 0) / len(ns):7.0%} "
              f"{sum(1 for x in ns_pos if x == 0) / len(ns_pos):13.0%}")
    print(f"\nAll models: {sum(all_n) / len(all_n):.2f} findings per review "
          f"(n={len(all_n)} parsed reviews), "
          f"{sum(1 for x in all_n if x == 0) / len(all_n):.0%} empty.")


if __name__ == "__main__":
    main()
