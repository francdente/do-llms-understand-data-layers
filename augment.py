"""
augment.py — rewrites each task's base implementation into multiple
language / framework / ORM variants using an LLM.

Also migrates existing flat-layout tasks (main.py at task root) into
the structured variants/ layout automatically.

Usage:
    uv run augment.py                          # all tasks, all variants
    uv run augment.py --task P1_test           # single task, all variants
    uv run augment.py --variant js_express_rawsql js_fastify_rawsql
"""
import argparse
import os
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

from dotenv import load_dotenv
from litellm import completion

# ---------------------------------------------------------------------------
# Variant registry
# Each entry: (variant_key, description, file_extension)
# The base variant (python_flask_rawsql) is the source — not generated.
# ---------------------------------------------------------------------------
VARIANTS = [
    ("python_flask_sqlalchemy",   "Python / Flask / SQLAlchemy ORM",                "py"),
    ("python_fastapi_rawsql",     "Python / FastAPI / raw SQL (sqlite3 stdlib)",     "py"),
    ("python_fastapi_sqlalchemy", "Python / FastAPI / SQLAlchemy ORM",               "py"),
    ("js_express_rawsql",         "JavaScript / Express / raw SQL (better-sqlite3)", "js"),
    ("js_express_sequelize",      "JavaScript / Express / Sequelize ORM",            "js"),
    ("js_fastify_rawsql",         "JavaScript / Fastify / raw SQL (better-sqlite3)", "js"),
    ("js_fastify_sequelize",      "JavaScript / Fastify / Sequelize ORM",            "js"),
    ("go_gin_rawsql",             "Go / Gin / raw SQL (database/sql + modernc.org/sqlite)", "go"),
    ("go_gin_gorm",               "Go / Gin / GORM (with sqlite driver)",            "go"),
    ("go_fiber_rawsql",           "Go / Fiber / raw SQL (database/sql + modernc.org/sqlite)", "go"),
    ("go_fiber_gorm",             "Go / Fiber / GORM (with sqlite driver)",          "go"),
]

BASE_VARIANT = "python_flask_rawsql"

REWRITE_PROMPT = """\
You are an expert backend software engineer.

Rewrite the application below from Python / Flask / raw SQL into {target}.

STRICT REQUIREMENTS — violating any of these invalidates the output:
1. Preserve every endpoint exactly: same URL paths, HTTP methods, query parameters,
   and JSON request/response shapes.
2. Preserve database access semantics without exception:
   - Every query must return the same rows as the original.
   - If the original explicitly deletes child rows before the parent (cascade),
     reproduce that deletion in the same order.
   - Every JOIN, subquery, or parent-existence filter on a read endpoint must be
     reproduced exactly — do not add or remove joins.
3. Use SQLite as the underlying database engine.
4. Output a single complete, runnable source file. No markdown fences, no prose.
   For Go targets: use a single main.go file with package main — do not split into packages.
5. Keep all the comments in the original file.
Database schema (SQLite):
<db_schema>
{db_schema}
</db_schema>

Original implementation (Python / Flask / raw SQL):
<original>
{original_code}
</original>

Output only the rewritten source file for {target}.
"""


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def find_base(task_path: Path) -> Path | None:
    """Return the base main.py, checking structured layout then flat layout."""
    structured = task_path / "variants" / BASE_VARIANT / "main.py"
    if structured.exists():
        return structured
    flat = task_path / "main.py"
    if flat.exists():
        return flat
    return None


def migrate_flat(task_path: Path, base_file: Path) -> Path:
    """Copy a flat main.py into variants/python_flask_rawsql/main.py."""
    dest = task_path / "variants" / BASE_VARIANT / "main.py"
    if not dest.exists():
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_text(base_file.read_text())
        print(f"    migrated → variants/{BASE_VARIANT}/main.py")
    return dest


def generate(source_code: str, schema: str, target: str, model: str, api_key: str) -> str:
    prompt = REWRITE_PROMPT.format(
        target=target,
        db_schema=schema,
        original_code=source_code,
    )
    response = completion(
        model=model,
        messages=[{"role": "user", "content": prompt}],
        temperature=0.0,
        api_key=api_key,
    )
    return response.choices[0].message.content.strip()


# ---------------------------------------------------------------------------
# Core
# ---------------------------------------------------------------------------

def _generate_variant(
    source_code: str,
    schema: str,
    task_name: str,
    variant_key: str,
    target_description: str,
    dest: Path,
    model: str,
    api_key: str,
) -> str:
    """Generate a single variant. Returns a status string for logging."""
    try:
        code = generate(source_code, schema, target_description, model, api_key)
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_text(code)
        return f"  ok    {task_name}/{variant_key}"
    except Exception as e:
        return f"  ERROR {task_name}/{variant_key} — {e}"


def process_task(
    task_path: Path,
    model: str,
    api_key: str,
    only_variants: list[str] | None,
    executor: ThreadPoolExecutor,
) -> list:
    """Submit variant generation jobs to the executor. Returns list of futures."""
    schema_path = task_path / "db_schema.sql"
    if not schema_path.exists():
        print(f"  SKIP {task_path.name} — no db_schema.sql")
        return []

    base_file = find_base(task_path)
    if base_file is None:
        print(f"  SKIP {task_path.name} — no base main.py found")
        return []

    # Migrate flat layout if needed
    if base_file.name == "main.py" and base_file.parent == task_path:
        base_file = migrate_flat(task_path, base_file)

    source_code = base_file.read_text()
    schema = schema_path.read_text()

    futures = []
    for variant_key, target_description, ext in VARIANTS:
        if only_variants and variant_key not in only_variants:
            continue

        dest = task_path / "variants" / variant_key / f"main.{ext}"
        if dest.exists():
            print(f"  skip  {task_path.name}/{variant_key} (exists)")
            continue

        fut = executor.submit(
            _generate_variant,
            source_code, schema, task_path.name,
            variant_key, target_description, dest,
            model, api_key,
        )
        futures.append(fut)

    return futures


def main() -> None:
    load_dotenv()

    parser = argparse.ArgumentParser()
    parser.add_argument("--tasks-dir", default="data/tasks",
                        help="Directory containing task subdirectories")
    parser.add_argument("--task",
                        help="Process only this task (directory name)")
    parser.add_argument("--variant", nargs="+", dest="variants",
                        help="Generate only these variant keys")
    parser.add_argument("--max-workers", type=int, default=4,
                        help="Max parallel LLM requests (default: 4)")
    args = parser.parse_args()

    model = os.environ["LLM_MODEL"]
    api_key = os.environ["LLM_API_KEY"]

    tasks_dir = Path(args.tasks_dir)

    if args.task:
        task_dirs = [tasks_dir / args.task]
    else:
        task_dirs = sorted(p for p in tasks_dir.iterdir() if p.is_dir())

    all_futures = []
    with ThreadPoolExecutor(max_workers=args.max_workers) as executor:
        for task_path in task_dirs:
            if not task_path.is_dir():
                print(f"not found: {task_path}")
                continue
            futures = process_task(task_path, model, api_key, args.variants, executor)
            all_futures.extend(futures)

        if all_futures:
            print(f"\n{len(all_futures)} variant(s) generating (max {args.max_workers} parallel)...")

        for fut in as_completed(all_futures):
            print(fut.result())


if __name__ == "__main__":
    main()
