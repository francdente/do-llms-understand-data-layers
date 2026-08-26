import argparse
import json
import os
from collections import defaultdict
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from threading import Lock

from dataclass_wizard import fromlist  # type: ignore
from dotenv import load_dotenv
import litellm

from src.core import Label, Scenario, ScenarioResult
from src.eval import (
    Count,
    Metrics,
    RunResult,
    count_to_dict,
    metrics_avg,
    metrics_to_dict,
    output_overall_results,
    output_results_per_run,
    output_results_per_variant,
    output_run_results,
)
from src.judge import execute_judge
from src.llm import LLMConfig, invoke_llm
from src.prompts import definitions
from src.prompts import instruction as inst
from src.utils import parse_json_lenient, save_results, strip_json_fences

LABEL_TO_DEFINITION = {
    "cascade_delete": definitions.CASCADE_DELETE,
    "stale_aggregate": definitions.STALE_AGGREGATE,
    "exposed_record": definitions.EXPOSED_RECORD,
}


@dataclass
class Job:
    task: Scenario
    prompt_type: str
    label: str | None
    task_llm_config: LLMConfig
    judge_llm_config: LLMConfig
    temperature: float
    reasoning_effort: str | None = None


@dataclass
class JobResult:
    scenario: Scenario
    result: ScenarioResult | None


def _run_scenario(job: Job) -> ScenarioResult | None:
    # prompt
    schema = job.task.db_schema
    code = job.task.code
    label = job.label

    match job.prompt_type:
        case "free":
            prompt = inst.INSTRUCTION_TEMPLATE_FREE.format(code=code, db_schema=schema)
        case "definition":
            assert label is not None
            prompt = inst.INSTRUCTION_TEMPLATE_DEFINITION.format(
                code=code,
                db_schema=schema,
                smell_definition=LABEL_TO_DEFINITION[label],
            )
        case _:
            raise Exception(f"Unknown prompt_type {job.prompt_type}!")

    # execute model
    llm_response = invoke_llm(
        prompt,
        job.task_llm_config.model,
        job.task_llm_config.api_key,
        job.task_llm_config.base_url,
        job.temperature,
        job.reasoning_effort,
    )

    if llm_response.finish_reason == "length":
        print(f"  WARNING: {job.task.name}/{job.task.variant} hit max_tokens (finish_reason=length)")

    content = llm_response.content or ""

    # parse / elaborate output
    if job.prompt_type == "free":
        result = execute_judge(job.task, content, job.judge_llm_config)
        return ScenarioResult(result, content, llm_response.reasoning, llm_response.finish_reason, llm_response.usage)
    else:
        try:
            parsed = parse_json_lenient(content)
            result = fromlist(Label, parsed)
            return ScenarioResult(result, content, llm_response.reasoning, llm_response.finish_reason, llm_response.usage)
        except (json.JSONDecodeError, ValueError) as e:
            print(f"ERROR parsing JSON: {e}")
            return None
        except Exception as e:
            print(f"Unknown error: {e}")
            return None


def main(
    jsonl_path: str,
    prompt_type: str,
    task_llm_config: LLMConfig,
    judge_llm_config: LLMConfig,
    results_dir: str,
    label: str | None = None,
    variant: str | None = None,
    runs: int = 3,
    max_workers: int = 10,
    temperature: float = 0.0,
    reasoning_effort: str | None = None,
):
    with open(jsonl_path, "r") as jsonl:
        lines = jsonl.readlines()
        dataset: list[Scenario] = [Scenario.from_json(line) for line in lines]  # type: ignore
        dataset = sorted(dataset, key=lambda e: e.name)
    jobs = [
        Job(task, prompt_type, label, task_llm_config, judge_llm_config, temperature, reasoning_effort)
        for task in dataset
        if (label is None or label in task.labels) and (variant is None or variant in task.variant)
    ]

    path_results_dir = Path(results_dir)
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    print(f"Starting execution! Timestamp {timestamp}\nSaving results in {path_results_dir}")

    # map[label, list[metrics] (one per run)]
    aggregate_metrics: dict[str, list[Metrics]] = dict()
    model_tag = task_llm_config.model.replace("/", "_")

    summary_path = path_results_dir / model_tag / prompt_type / timestamp

    summary = {
        "model": task_llm_config.model,
        "prompt_type": prompt_type,
        "label": label,
        "timestamp": timestamp,
        "runs": [],
        "aggregate": {},
    }

    for run in range(runs):
        print(f"\n[run {run + 1}] Submitting {len(jobs)} jobs...")

        run_results: list[RunResult] = []
        # map[variant_name, map[label, count]]
        count_by_variant: dict[str, dict[str, Count]] = dict()
        # map[label, count]
        count_by_run: dict[str, Count] = dict()

        print_lock = Lock()

        def _run_job(job: Job) -> JobResult:
            task_result = _run_scenario(job)
            return JobResult(job.task, task_result)

        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            futures = {executor.submit(_run_job, job): job for job in jobs}
            for future in as_completed(futures):
                job_result = future.result()
                scenario = job_result.scenario
                scenario_result = job_result.result

                with print_lock:
                    status = "ok" if scenario_result else "ERROR"
                    print(f"  done  {scenario.name}/{scenario.variant}  [{status}]")

                if not scenario_result:
                    continue

                # save results
                out_dir = summary_path / scenario.name / f"run_{run}"
                out_dir.mkdir(parents=True, exist_ok=True)
                save_results(out_dir, scenario, scenario_result)

                # compute count
                results_dict = {label.name: label for label in scenario_result.labels}
                for ground_truth in scenario.ground_truth:
                    if label is not None and ground_truth.name != label:
                        continue

                    result_label = results_dict.get(ground_truth.name)
                    if result_label is None:
                        print(f"  {scenario.name} {scenario.variant} - cannot find entry for {ground_truth.name}!")
                        continue

                    count = Count()
                    count.check(ground_truth, result_label)
                    run_result = RunResult(scenario.name, ground_truth.name, scenario.variant, count)
                    run_results.append(run_result)

                    entry_by_run = count_by_run.setdefault(ground_truth.name, Count())
                    entry_by_run.add(count)

                    entry_by_variant = count_by_variant.setdefault(
                        scenario.variant,
                        defaultdict(),
                    ).setdefault(ground_truth.name, Count())
                    entry_by_variant.add(count)

        output_run_results(run_results)
        output_results_per_variant(count_by_variant)
        output_results_per_run(count_by_run)

        run_summary = {
            "run": run,
            "details": [
                {"task": r.name, "label": r.label, "variant": r.variant, **count_to_dict(r.count)}
                for r in sorted(run_results, key=lambda r: f"{r.name}_{r.label}_{r.variant}")
            ],
            "by_variant": {
                v: {l: metrics_to_dict(c.to_metrics()) for l, c in labels.items()}
                for v, labels in count_by_variant.items()
            },
            "by_label": {l: metrics_to_dict(c.to_metrics()) for l, c in count_by_run.items()},
        }
        summary["runs"].append(run_summary)

        for label_key in count_by_run.keys():
            as_metrics = count_by_run[label_key].to_metrics()
            aggregate_metrics.setdefault(label_key, []).append(as_metrics)

    output_overall_results(aggregate_metrics)

    for label_key, m_list in aggregate_metrics.items():
        summary["aggregate"][label_key] = metrics_to_dict(metrics_avg(m_list))

    metrics_file = summary_path / "metrics.json"
    metrics_file.parent.mkdir(parents=True, exist_ok=True)
    with open(metrics_file, "w") as f:
        json.dump(summary, f, indent=2)
    print(f"\nMetrics saved → {metrics_file}")


if __name__ == "__main__":
    load_dotenv()
    parser = argparse.ArgumentParser()

    # Required
    parser.add_argument("--tasks", required=True, help="The absolute path to the tasks JSONL")
    parser.add_argument("--label", default=None, help="Label to evaluate (required for prompt definition, e.g. cascade_delete)")
    parser.add_argument(
        "--prompt-type",
        choices=["free", "definition"],
        default="definition",
        help="Prompt variant (default: definition)",
    )
    parser.add_argument("--temperature", type=float, default=0.0, help="temperature for the llm model")
    parser.add_argument("--reasoning-effort", choices=["none", "low", "medium", "high"], default=None, help="reasoning effort for supported models (e.g. GPT-5.4-mini)")

    # Filters
    parser.add_argument("--variant", default=None, help="Evaluate a single variant key (e.g. python_flask_rawsql)")

    # Misc
    parser.add_argument("--runs", type=int, default=1, help="Number of runs (default: 1)")
    parser.add_argument("--max-workers", type=int, default=4, help="Max parallel LLM requests (default: 4)")
    parser.add_argument(
        "--results-dir",
        default="data/results",
        help="Directory to save raw LLM outputs (default: data/results)",
    )

    args = parser.parse_args()

    if args.prompt_type == "definition" and not args.label:
        parser.error("--label is required for --prompt-type definition")

    tasks_llm_config = LLMConfig(
        os.environ["LLM_MODEL"],
        os.environ["LLM_API_KEY"],
        os.getenv("LLM_BASE_URL"),
    )
    judge_llm_config = LLMConfig(
        os.environ["LLM_JUDGE_MODEL"],
        os.environ["LLM_JUDGE_API_KEY"],
        os.getenv("LLM_JUDGE_BASE_URL"),
    )

    main(
        args.tasks,
        args.prompt_type,
        tasks_llm_config,
        judge_llm_config,
        args.results_dir,
        args.label,
        args.variant,
        args.runs,
        args.max_workers,
        args.temperature,
        args.reasoning_effort,
    )
