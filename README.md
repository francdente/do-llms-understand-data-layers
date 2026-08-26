# Do LLMs Understand Backend Data-Layer Semantics?

Reproducibility package for the paper:

> **Do LLMs Understand Backend Data-Layer Semantics? A Controlled Study of Cross-Endpoint Issues**
>
> EMNLP 2026

## Overview

This repo evaluates whether Large Language Models can detect **semantic data-layer issues** in database-backed backend applications. These are latent bugs that require cross-endpoint reasoning over persistent state -- they do not crash the program, but violate intended data semantics.

We study three issue categories, each targeting a distinct dimension of persistent-state semantics:

| Issue | Dimension | Description |
|---|---|---|
| `cascade_delete` | Retention | A delete cascades to child rows that should remain historically observable |
| `stale_aggregate` | Consistency | A denormalized aggregate becomes stale after writes to its source data |
| `exposed_record` | Visibility | An endpoint returns records outside their intended visibility scope |

Models are evaluated under two prompting regimes:
- **FREE**: open-ended code review (no issue definition provided)
- **DEF**: targeted detection (issue definition provided)

## Repository Structure

```
.
├── main.py                  # Main evaluation pipeline (Sec. 3: Methodology)
├── augment.py               # LLM-based task augmentation (Appendix A)
├── generate_jsonl.py        # Converts task directories to scenarios.jsonl
├── src/
│   ├── core.py              # Data classes (Scenario, Label, ScenarioResult)
│   ├── llm.py               # LLM invocation via litellm
│   ├── eval.py              # Metrics computation (P/R/F1)
│   ├── judge.py             # LLM judge for post-hoc validation (Appendix B)
│   ├── utils.py             # JSON parsing, result saving
│   └── prompts/
│       ├── instruction.py   # Prompt templates: FREE and DEF (Appendix C.2)
│       ├── definitions.py   # Issue definitions for DEF prompt (Appendix C.2)
│       └── judge.py         # Judge prompt template (Appendix B)
├── data/
│   ├── results.zip          # Pre-computed results for all models, incl. the
│   │                        #   temperature-0.7 repeat runs (see Quick Reproduction)
│   ├── results_ablation_paraphrase/  # Paraphrased DEF results (Appendix G)
│   ├── results_ablation_neutral/     # Neutrality-cue DEF results (Appendix E.1)
│   └── tasks/               # Evaluation benchmark (Sec. 3, Appendix A)
│       ├── scenarios.jsonl   # Pre-built JSONL with all 216 instances
│       ├── CD_short_P1_test/ # Example task directory
│       │   ├── db_schema.sql
│       │   ├── gt.json       # Ground truth labels
│       │   ├── main.py       # Base implementation (Python/Flask/raw SQL)
│       │   └── variants/     # 12 augmented variants
│       │       ├── python_flask_sqlalchemy/
│       │       ├── python_fastapi_rawsql/
│       │       ├── js_express_rawsql/
│       │       ├── go_gin_gorm/
│       │       └── ...
│       └── ...
├── manual_review/              # Manual FREE annotations (Appendix B)
│   ├── openai_gpt-5.4_annotated.csv       # Author F annotations (TRUE/FALSE)
│   ├── ...                                 # One file per model (5 models)
│   └── inter_annotations/                 # Second-author (D) annotations for 4 models
│       ├── annotations-gemma.csv
│       ├── annotations-kimi.csv
│       ├── annotations-deepseek.csv
│       └── annotations-qwen.csv
├── reproduce_table.py          # Reproduce main results table (Sec. 4, Table 2)
├── compute_free_from_annotations.py  # Recompute FREE metrics from manual annotations
├── compute_judge_agreement.py  # Inter-annotator agreement: manual vs LLM judge
├── compute_task_stats.py       # Benchmark statistics (Appendix A)
├── compute_review_stats.py     # Evaluation result statistics (Appendix B, F)
├── compare_paraphrase_ablation.py  # Prompt sensitivity ablation (Appendix G)
├── run_ablation_paraphrase.py  # Run paraphrased DEF definitions
├── compute_temperature_ablation.py  # Temperature sensitivity, T=0 vs T=0.7 (Appendix G)
├── run_ablation_neutral.py     # Run DEF with a neutrality cue (Appendix E.1)
├── compare_neutral_ablation.py # Neutrality ablation table (Appendix E.1)
├── free_review_stats.py        # Findings per FREE review (Appendix H)
├── per_task_table.py           # Per-base-task correct outcomes (Appendix I)
├── compare_base_augmented.py   # Hand-crafted vs augmented tasks + DiD (Appendix J)
├── bootstrap_ci.py             # Cluster bootstrap CIs over base tasks (Appendix K)
├── compute_inter_annotator_agreement.py  # Inter-annotator agreement: Author F vs D vs Judge
├── human-results.py            # Human validation analysis (Appendix D)
├── pyproject.toml
└── .env.template               # Environment variable template
```

## Benchmark Design

**18 base tasks** (6 per issue category: 3 positive, 3 negative) are each augmented into **12 variants** across:
- **3 languages**: Python, JavaScript, Go
- **6 frameworks**: Flask, FastAPI, Express, Fastify, Gin, Fiber
- **2 data-access patterns**: raw SQL, ORM (SQLAlchemy, Sequelize, GORM)

This yields **216 evaluation instances** total. Each instance consists of a database schema (`db_schema.sql`), backend code, and ground truth labels (`gt.json`).

## Setup

### Requirements

- Python >= 3.12
- [uv](https://docs.astral.sh/uv/)

### Installation
Clone the repository and install the dependencies via `uv`.

```bash
# Install dependencies with uv
uv sync
```

### Configuration

Copy the environment template and fill in your API keys:

```bash
cp .env.template .env
```

Edit `.env` with your configuration:

```bash
# Task LLM (the model being evaluated)
LLM_API_KEY="your-api-key"
LLM_MODEL="openai/gpt-5.4"          # litellm model identifier

# Output settings
LLM_MAX_TOKENS=64000
LLM_DISABLE_THINKING=true
```

Model identifiers follow the [litellm naming convention](https://docs.litellm.ai/docs/providers), e.g.:
- `openai/gpt-5.4`
- `together_ai/deepseek-ai/DeepSeek-V3.1`
- `ollama/qwen3.6:35b` (with `LLM_BASE_URL=http://localhost:11434`)

## Running Experiments

### 1. Generate the scenarios file (optional)

The `scenarios.jsonl` file is pre-built and included. To regenerate it from the task directories:

```bash
uv run generate_jsonl.py --tasks data/tasks --output data/tasks/scenarios.jsonl
```

### 2. Run evaluation

**DEF prompt** (targeted detection, one label at a time):

```bash
uv run main.py \
  --tasks data/tasks/scenarios.jsonl \
  --prompt-type definition \
  --label cascade_delete \
  --max-workers 4
```

**FREE prompt** (open-ended review):

```bash
uv run main.py \
  --tasks data/tasks/scenarios.jsonl \
  --prompt-type free \
  --max-workers 4
```

### CLI Options

| Flag | Description | Default |
|---|---|---|
| `--tasks` | Path to `scenarios.jsonl` | (required) |
| `--prompt-type` | `free` or `definition` | `definition` |
| `--label` | Issue label (required for `definition`) | `None` |
| `--runs` | Number of evaluation runs | `1` |
| `--max-workers` | Parallel LLM requests | `4` |
| `--temperature` | LLM temperature | `0.0` |
| `--reasoning-effort` | `none`, `low`, `medium`, `high` (for supported models) | `None` |
| `--variant` | Evaluate a single variant (e.g., `python_flask_rawsql`) | `None` |
| `--results-dir` | Output directory for results | `data/results` |

### 3. Output

Results are saved to `data/results/<model>/<prompt_type>/<timestamp>/`:

```
data/results/openai_gpt-5.4/definition/20260513_011923/
├── metrics.json              # Aggregate P/R/F1 per label and run
├── CD_short_P1_test/
│   └── run_0/
│       ├── python_flask_rawsql.json       # Parsed labels
│       ├── python_flask_rawsql_meta.json  # Usage metadata
│       └── python_flask_rawsql_raw.txt    # Raw LLM output
└── ...
```

The `metrics.json` file contains per-run and aggregate precision, recall, and F1 scores.

## FREE Output Classification

FREE outputs are classified by **manual annotation** rather than an LLM judge. One of the authors (Author F) manually classified all 1,080 FREE outputs (216 instances x 5 models) against the ground truth. A second author (Author D) independently re-annotated 864 instances (4 models), yielding 99.7% agreement (Cohen's kappa = 0.99). The annotated spreadsheets are in `manual_review/`:

- `<model>_annotated.csv` — Author F's annotations with task, variant, label, ground truth, raw model output, judge output, and `my_classification` column (TRUE/FALSE)
- `inter_annotations/<model>.csv` — Both authors' annotations (`F_classification`, `D_classification`) for 4 models

A post-hoc comparison against an LLM judge (GPT-5.4) shows 94.1% agreement with Author F (Cohen's kappa = 0.84) and 92.2% with Author D (kappa = 0.78), confirming that this step could be automated for larger-scale evaluations. See `compute_judge_agreement.py` and `compute_inter_annotator_agreement.py`.

## Task Augmentation

To generate new variants from base tasks using an LLM:

```bash
uv run augment.py                                  # All tasks, all variants
uv run augment.py --task CD_short_P1_test          # Single task
uv run augment.py --variant js_express_rawsql      # Single variant
```

This uses the LLM configured in `.env` to rewrite the base Python/Flask/raw SQL implementation into other language/framework/ORM combinations.

## Reproducing Paper Results

### Quick reproduction (no API keys needed)

`data/results.zip` contains the exact model outputs and `metrics.json` files used to produce every number in the paper. To reproduce the tables and statistics without re-running any LLM:

```bash
unzip data/results.zip -d data

# Table 2: main results (DEF rows from metrics.json, FREE rows from manual annotations)
uv run reproduce_table.py #compute full table by using LLM-judge outputs for FREE prompting
uv run compute_free_from_annotations.py #compute FREE scores by using authors' annotations for FREE prompting

# Benchmark statistics (Tables 3-4)
uv run compute_task_stats.py

# JSON exclusions (Table 5), DEF variant breakdowns (Appendix F, Tables 8-9)
uv run compute_review_stats.py

# Inter-annotator agreement: manual vs LLM judge (Appendix B)
uv run compute_judge_agreement.py

# Inter-annotator agreement: Author F vs Author D vs LLM judge (Appendix B)
uv run compute_inter_annotator_agreement.py

# Prompt sensitivity ablation (Appendix G, Tables 10-11)
uv run compare_paraphrase_ablation.py
uv run compute_temperature_ablation.py

# Neutrality ablation (Appendix E.1, Table 7)
uv run compare_neutral_ablation.py

# FREE review content: findings per review, empty-review rates (Appendix H, Table 12)
uv run free_review_stats.py

# Per-base-task correct outcomes (Appendix I, Table 13)
uv run per_task_table.py

# Hand-crafted vs augmented tasks + difference-in-differences (Appendix J, Tables 14-15)
uv run compare_base_augmented.py

# 95% cluster-bootstrap CIs over the 18 base tasks (Appendix K, Table 16)
uv run bootstrap_ci.py
```

### Full reproduction (requires API keys)

To re-run the experiments from scratch, follow the [Setup](#setup) and [Running Experiments](#running-experiments) sections above. New results will be saved under `data/results/` with fresh timestamps.

### Paper artifact mapping

The table below maps each paper artifact to the code and data that produced it.

| Paper Section | Artifact | Script / Data |
|---|---|---|
| Sec. 3 — Issue Taxonomy (Table 1) | Issue definitions | `src/prompts/definitions.py` |
| Sec. 3 — Task Construction | 18 base tasks | `data/tasks/*/main.py`, `db_schema.sql`, `gt.json` |
| Sec. 3 — Task Augmentation | 216 instances (12 variants each) | `augment.py`, `data/tasks/*/variants/` |
| Sec. 3 — Evaluation | DEF and FREE detection runs | `main.py --prompt-type definition\|free` |
| Sec. 4 — Results (Table 2) | P/R/F1 per model and category | `reproduce_table.py` (DEF), `compute_free_from_annotations.py` (FREE) |
| Appendix A — Task Construction | Base task statistics (Table 3) | `compute_task_stats.py` |
| Appendix A — Augmentation | Variant LoC statistics (Table 4) | `compute_task_stats.py` |
| Appendix B — Evaluation Details | Scoring pipeline, metrics | `src/eval.py` |
| Appendix B — FREE annotation | Manual classifications (1,080 instances) | `manual_review/*_annotated.csv` |
| Appendix B — Inter-annotator agreement | Human F vs D agreement | `compute_inter_annotator_agreement.py` |
| Appendix B — LLM judge validation | Human vs LLM judge agreement | `compute_judge_agreement.py`, `compute_inter_annotator_agreement.py` |
| Appendix B — JSON errors (Table 5) | DEF parsing exclusions | `compute_review_stats.py` |
| Appendix C.1 — Augmentation Prompt | Rewrite prompt | `augment.py` (see `REWRITE_PROMPT`) |
| Appendix C.2 — Detection Prompts | FREE and DEF prompts | `src/prompts/instruction.py` |
| Appendix C.2 — Issue Definitions | Definitions injected in DEF | `src/prompts/definitions.py` |
| Appendix D — Human Validation | Expert detection rates (Table 6) | `human-results.py` |
| Appendix E — Compliance Bias | Qualitative analysis | `data/results/` (raw model outputs) |
| Appendix E.1 — Neutrality Ablation | Neutrality cue on DEF/cascade_delete (Table 7) | `run_ablation_neutral.py`, `compare_neutral_ablation.py`, `data/results_ablation_neutral/` |
| Appendix F — Variant Breakdown | F1 by language/access pattern (Tables 8-9) | `compute_free_from_annotations.py` (FREE), `compute_review_stats.py` (DEF) |
| Appendix G — Prompt Sensitivity | Paraphrased DEF ablation (Table 10) | `run_ablation_paraphrase.py`, `compare_paraphrase_ablation.py` |
| Appendix G — Temperature Sensitivity | T=0 vs T=0.7, 3 runs (Table 11) | `compute_temperature_ablation.py` |
| Appendix H — FREE Review Content | Findings per review, empty-review rates (Table 12) | `free_review_stats.py` |
| Appendix I — Per-Base-Task Results | Correct outcomes per base task (Table 13) | `per_task_table.py` |
| Appendix J — Hand-Crafted vs Augmented | Base/augmented split + DiD + Fisher tests (Tables 14-15) | `compare_base_augmented.py` |
| Appendix K — Statistical Significance | Cluster bootstrap CIs over base tasks (Table 16) | `bootstrap_ci.py` |
| Appendix L — Representative Repairs | Qualitative | — |

### Models evaluated

| Model | Provider | Identifier |
|---|---|---|
| GPT-5.4 | OpenAI | `openai/gpt-5.4` |
| DeepSeek-V3.1 | Together AI | `together_ai/deepseek-ai/DeepSeek-V3.1` |
| Kimi-K2.6 | Together AI | `together_ai/moonshotai/Kimi-K2.6` |
| Gemma-4-31B-IT | Google (via Ollama/vLLM) | `google/gemma-4-31b-it` |
| Qwen3.5-397B-A17B | Alibaba (via Together AI) | `together_ai/Qwen/Qwen3.5-397B-A17B` |

All experiments use temperature 0 and reasoning disabled for reproducibility.

### AI-disclosure
The code in this repository was developed with AI assistance.

## License

See [LICENSE](LICENSE) for details.
