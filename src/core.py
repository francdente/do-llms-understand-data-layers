from dataclasses import dataclass, field

from dataclass_wizard import JSONWizard


@dataclass
class Label:
    name: str
    detected: bool
    description: str


@dataclass
class Scenario(JSONWizard):
    name: str
    variant: str
    db_schema: str
    labels: list[str]
    ground_truth: list[Label]
    code: str


@dataclass
class ScenarioResult:
    labels: list[Label]
    raw_output: str | None = None
    reasoning: str | None = None
    finish_reason: str | None = None
    usage: dict | None = field(default=None)
