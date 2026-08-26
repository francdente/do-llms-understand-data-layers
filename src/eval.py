from collections import defaultdict
from dataclasses import dataclass
from typing import Self

from src.core import Label


@dataclass
class Metrics:
    # float allows to compute the mean using the same class
    tp: float
    fp: float
    tn: float
    fn: float
    precision: float
    recall: float
    f1: float
    accuracy: float

    def __str__(self) -> str:
        return (
            f"TP: {self.tp:.2f}\tFP: {self.fp:.2f}\tTN: {self.tn:.2f}\tFN: {self.fn:.2f}\n"
            f"Prec.: {self.precision:.2f}\tRecall: {self.recall:.2f}\tF1: {self.f1:.2f}\tAccuracy: {self.accuracy:.2f}"
        )


@dataclass
class Count:
    tp: int = 0
    fp: int = 0
    tn: int = 0
    fn: int = 0

    def check(self, gt_label: Label, llm_label: Label) -> None:
        assert gt_label.name == llm_label.name

        if gt_label.detected and llm_label.detected:
            self.tp += 1
        elif not gt_label.detected and not llm_label.detected:
            self.tn += 1
        elif not gt_label.detected and llm_label.detected:
            self.fp += 1
        else:
            self.fn += 1

    def add(self, count: Self) -> None:
        self.tp += count.tp
        self.fp += count.fp
        self.tn += count.tn
        self.fn += count.fn

    def to_metrics(self) -> Metrics:
        tp, fp, tn, fn = self.tp, self.fp, self.tn, self.fn
        total = tp + fp + tn + fn
        precision = tp / (tp + fp) if (tp + fp) > 0 else 0.0
        recall = tp / (tp + fn) if (tp + fn) > 0 else 0.0
        f1 = 2 * precision * recall / (precision + recall) if (precision + recall) > 0 else 0.0
        accuracy = (tp + tn) / total if total > 0 else 0.0
        return Metrics(tp, fp, tn, fn, precision, recall, f1, accuracy)

    def __str__(self) -> str:
        return f"TP: {self.tp}\tFP: {self.fp}\tTN: {self.tn}\tFN: {self.fn}"


@dataclass
class RunResult:
    name: str
    label: str
    variant: str
    count: Count


def metrics_avg(metrics: list[Metrics]) -> Metrics:
    tp = 0
    fp = 0
    tn = 0
    fn = 0
    precision = 0
    recall = 0
    f1 = 0
    accuracy = 0

    for m in metrics:
        tp += m.tp
        fp += m.fp
        tn += m.tn
        fn += m.fn
        precision += m.precision
        recall += m.recall
        f1 += m.f1
        accuracy += m.accuracy

    num = len(metrics)
    tp /= num
    fp /= num
    tn /= num
    fn /= num
    precision /= num
    recall /= num
    f1 /= num
    accuracy /= num

    return Metrics(tp, fp, tn, fn, precision, recall, f1, accuracy)


def metrics_to_dict(m: Metrics) -> dict:
    return {
        "tp": m.tp, "fp": m.fp, "tn": m.tn, "fn": m.fn,
        "precision": m.precision, "recall": m.recall,
        "f1": m.f1, "accuracy": m.accuracy,
    }


def count_to_dict(c: Count) -> dict:
    return {
        "tp": c.tp, "fp": c.fp, "tn": c.tn, "fn": c.fn,
    }


def output_run_results(run_results: list[RunResult]) -> None:
    for result in sorted(run_results, key=lambda r: f"{r.name}_{r.label}_{r.variant}"):
        print(f"\n  ### {result.name} | {result.label} | {result.variant}:\t{result.count}")


def output_results_per_variant(count_by_variant: dict[str, dict[str, Count]]) -> None:
    print("\n==== Aggregate by variant")
    for variant in count_by_variant.keys():
        print(f"\n### {variant}")
        for label in count_by_variant[variant].keys():
            as_metrics = count_by_variant[variant][label].to_metrics()
            print(f"\n{label}\n{as_metrics}")
    print("\n====")


def output_results_per_run(count_by_run: dict[str, Count]) -> None:
    print("\n==== Aggregate by run")
    for label in count_by_run.keys():
        print(f"\n### {label}")
        as_metrics = count_by_run[label].to_metrics()
        print(as_metrics)
    print("\n====")


def output_overall_results(aggregate_metrics: dict[str, list[Metrics]]) -> None:
    aggregate: dict[str, Metrics] = defaultdict()
    print("\n\n==== # Overall results")
    for label in aggregate_metrics.keys():
        metric_list = aggregate_metrics[label]
        avg = metrics_avg(metric_list)
        aggregate.setdefault(label, avg)
        print(f"\n## {label}")
        print(avg)
    print("\n====")
