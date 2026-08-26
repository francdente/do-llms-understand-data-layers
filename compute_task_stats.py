"""
compute_task_stats.py — Compute statistics over ALL 216 evaluation instances
(18 base tasks × 12 variants) for the paper appendix.

Outputs:
  - Base task stats (schema tables, columns, FKs, endpoints, LoC)
  - Per-variant stats (LoC across languages/frameworks/ORMs)
  - Aggregated stats across all 216 instances
"""

import re
from pathlib import Path
from collections import defaultdict

TASKS_DIR = Path("data/tasks")

VARIANT_KEYS = [
    "python_flask_rawsql",
    "python_flask_sqlalchemy",
    "python_fastapi_rawsql",
    "python_fastapi_sqlalchemy",
    "js_express_rawsql",
    "js_express_sequelize",
    "js_fastify_rawsql",
    "js_fastify_sequelize",
    "go_gin_rawsql",
    "go_gin_gorm",
    "go_fiber_rawsql",
    "go_fiber_gorm",
]


def count_tables(schema: str) -> int:
    return len(re.findall(r"CREATE\s+TABLE", schema, re.IGNORECASE))


def count_columns(schema: str) -> int:
    total = 0
    for match in re.finditer(
        r"CREATE\s+TABLE\s+\w+\s*\((.*?)\)", schema, re.IGNORECASE | re.DOTALL
    ):
        body = match.group(1)
        for line in body.split(","):
            line = line.strip()
            if not line:
                continue
            # skip constraints
            if re.match(
                r"(PRIMARY\s+KEY|FOREIGN\s+KEY|UNIQUE|CHECK|CONSTRAINT)",
                line,
                re.IGNORECASE,
            ):
                continue
            total += 1
    return total


def count_fks(schema: str) -> int:
    return len(re.findall(r"FOREIGN\s+KEY", schema, re.IGNORECASE))


def count_endpoints(code: str) -> int:
    # Python: @app.route / @app.get / @app.post / @app.put / @app.delete
    py = len(re.findall(r"@app\.(route|get|post|put|delete|patch)", code))
    # JS Express/Fastify: app.get / app.post / router.get / fastify.get etc.
    js = len(
        re.findall(
            r"(app|router|fastify)\.(get|post|put|delete|patch)\s*\(",
            code,
            re.IGNORECASE,
        )
    )
    # Go Gin/Fiber: r.GET / r.POST / app.Get / app.Post
    go = len(
        re.findall(
            r"\.(GET|POST|PUT|DELETE|PATCH|Get|Post|Put|Delete|Patch)\s*\(",
            code,
        )
    )
    return py + js + go


def loc(code: str) -> int:
    return len([l for l in code.splitlines() if l.strip()])


def main():
    task_dirs = sorted(
        [d for d in TASKS_DIR.iterdir() if d.is_dir()],
        key=lambda d: d.name,
    )

    # --- Base task stats ---
    print("=" * 60)
    print("BASE TASK STATS (18 tasks, Python/Flask/raw SQL)")
    print("=" * 60)

    base_stats = {
        "tables": [],
        "columns": [],
        "fks": [],
        "endpoints": [],
        "loc": [],
    }

    for td in task_dirs:
        schema_file = td / "db_schema.sql"
        code_file = td / "main.py"
        if not schema_file.exists() or not code_file.exists():
            continue

        schema = schema_file.read_text()
        code = code_file.read_text()

        t = count_tables(schema)
        c = count_columns(schema)
        f = count_fks(schema)
        e = count_endpoints(code)
        l = loc(code)

        base_stats["tables"].append(t)
        base_stats["columns"].append(c)
        base_stats["fks"].append(f)
        base_stats["endpoints"].append(e)
        base_stats["loc"].append(l)

        print(f"  {td.name:30s}  tables={t}  cols={c}  fks={f}  endpoints={e}  loc={l}")

    print()
    for key, vals in base_stats.items():
        print(
            f"  {key:20s}  min={min(vals):3d}  max={max(vals):3d}  "
            f"avg={sum(vals)/len(vals):.1f}  median={sorted(vals)[len(vals)//2]}"
        )

    # --- All 216 variant stats ---
    print()
    print("=" * 60)
    print("ALL VARIANT STATS (216 instances)")
    print("=" * 60)

    all_loc = []
    loc_by_lang = defaultdict(list)
    loc_by_framework = defaultdict(list)
    loc_by_access = defaultdict(list)
    loc_by_variant = defaultdict(list)

    for td in task_dirs:
        schema_file = td / "db_schema.sql"
        if not schema_file.exists():
            continue

        for vk in VARIANT_KEYS:
            # determine file extension
            if vk.startswith("python"):
                ext = "py"
                lang = "Python"
            elif vk.startswith("js"):
                ext = "js"
                lang = "JavaScript"
            elif vk.startswith("go"):
                ext = "go"
                lang = "Go"
            else:
                continue

            # framework
            parts = vk.split("_")
            framework = parts[1]  # flask, fastapi, express, fastify, gin, fiber
            access = parts[2]     # rawsql, sqlalchemy, sequelize, gorm

            variant_dir = td / "variants" / vk
            code_file = variant_dir / f"main.{ext}"

            if not code_file.exists():
                # base variant lives at task root
                if vk == "python_flask_rawsql":
                    code_file = td / "main.py"
                if not code_file.exists():
                    print(f"  MISSING: {td.name}/{vk}")
                    continue

            code = code_file.read_text()
            l = loc(code)

            all_loc.append(l)
            loc_by_lang[lang].append(l)
            loc_by_framework[f"{lang}/{framework}"].append(l)
            loc_by_access[access].append(l)
            loc_by_variant[vk].append(l)

    print(f"\n  Total instances: {len(all_loc)}")
    print(
        f"  LoC across all:  min={min(all_loc)}  max={max(all_loc)}  "
        f"avg={sum(all_loc)/len(all_loc):.1f}  median={sorted(all_loc)[len(all_loc)//2]}"
    )

    print("\n  By language:")
    for lang in sorted(loc_by_lang):
        vals = loc_by_lang[lang]
        print(
            f"    {lang:15s}  n={len(vals):3d}  "
            f"min={min(vals):3d}  max={max(vals):3d}  avg={sum(vals)/len(vals):.1f}"
        )

    print("\n  By framework:")
    for fw in sorted(loc_by_framework):
        vals = loc_by_framework[fw]
        print(
            f"    {fw:25s}  n={len(vals):3d}  "
            f"min={min(vals):3d}  max={max(vals):3d}  avg={sum(vals)/len(vals):.1f}"
        )

    print("\n  By data access:")
    for acc in sorted(loc_by_access):
        vals = loc_by_access[acc]
        print(
            f"    {acc:15s}  n={len(vals):3d}  "
            f"min={min(vals):3d}  max={max(vals):3d}  avg={sum(vals)/len(vals):.1f}"
        )

    print("\n  By variant (all 12):")
    for vk in VARIANT_KEYS:
        vals = loc_by_variant[vk]
        print(
            f"    {vk:35s}  n={len(vals):3d}  "
            f"min={min(vals):3d}  max={max(vals):3d}  avg={sum(vals)/len(vals):.1f}"
        )


if __name__ == "__main__":
    main()
