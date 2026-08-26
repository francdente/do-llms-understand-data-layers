import json
import re
from pathlib import Path

from dataclass_wizard import asdict  # type: ignore

from src.core import Scenario, ScenarioResult


def strip_json_fences(text: str) -> str:
    m = re.match(r"^```(?:json)?\s*\n(.*?)```\s*$", text.strip(), re.DOTALL)
    return m.group(1).strip() if m else text


def parse_json_lenient(text: str) -> list:
    """Parse the first JSON array from text, ignoring trailing content."""
    cleaned = strip_json_fences(text)
    try:
        return json.loads(cleaned)
    except json.JSONDecodeError:
        decoder = json.JSONDecoder()
        obj, _ = decoder.raw_decode(cleaned)
        return obj


def save_results(
    out_dir: Path,
    scenario: Scenario,
    scenario_result: ScenarioResult,
) -> None:
    out_file = out_dir / f"{scenario.variant}.json"

    with open(out_file, "w") as f:
        json.dump(
            [asdict(label) for label in scenario_result.labels],
            f,
            indent=2,
        )

    if scenario_result.raw_output is not None:
        raw_file = out_dir / f"{scenario.variant}_raw.txt"
        with open(raw_file, "w") as f:
            f.write(scenario_result.raw_output)

    if scenario_result.reasoning is not None:
        reasoning_file = out_dir / f"{scenario.variant}_reasoning.txt"
        with open(reasoning_file, "w") as f:
            f.write(scenario_result.reasoning)

    meta = {}
    if scenario_result.finish_reason is not None:
        meta["finish_reason"] = scenario_result.finish_reason
    if scenario_result.usage is not None:
        meta["usage"] = scenario_result.usage
    if meta:
        meta_file = out_dir / f"{scenario.variant}_meta.json"
        with open(meta_file, "w") as f:
            json.dump(meta, f, indent=2)
