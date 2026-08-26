import argparse
import json
from pathlib import Path

from dataclass_wizard import asdict, fromlist  # type: ignore

from src.core import Label, Scenario


def main(tasks_dir: str, output_file: str) -> None:
    """
    Load all tasks from a directory into a JSONL file.

    Each task folder may contain:
      - main.py          → emits one entry with variant="base"
      - variants/<name>/ → emits one entry per variant (using the single file inside)
      - db_schema.sql    → shared across all entries for that task
      - gt.json          → shared across all entries for that task

    Args:
        tasks_dir:   Path to the root `tasks` directory.
        output_file: Destination .jsonl file path.
    """
    tasks_path = Path(tasks_dir)

    if not tasks_path.is_dir():
        raise NotADirectoryError(f"tasks directory not found: {tasks_path}")

    entries: list[Scenario] = []

    for task_dir in sorted(tasks_path.iterdir()):
        if not task_dir.is_dir():
            continue

        # 0. task_name
        task_name = task_dir.name

        # 0. schema & ground truth
        db_schema_path = task_dir / "db_schema.sql"
        gt_path = task_dir / "gt.json"

        if not db_schema_path.exists():
            raise FileNotFoundError(f"db_schema.sql is missing! {task_name}")
        db_schema = db_schema_path.read_text(encoding="utf-8")

        if not gt_path.exists():
            raise FileNotFoundError(f"gt.json is missing! {task_name}")
        ground_truth = gt_path.read_text(encoding="utf-8")
        ground_truth_json = json.loads(ground_truth)

        # 1. variants
        variants_dir = task_dir / "variants"
        if variants_dir.is_dir():
            for variant_dir in sorted(variants_dir.iterdir()):
                if not variant_dir.is_dir():
                    continue

                # Each variant folder contains exactly one file of content
                variant_files = [
                    f
                    for f in variant_dir.iterdir()
                    if f.is_file() and not f.name.startswith(".")
                ]

                if not variant_files:
                    continue  # empty variant – skip

                if len(variant_files) > 1:
                    variant_files.sort()
                    print(
                        "Warning! More than one variant in the folder, taking first! {}",
                        variant_files[0].name,
                    )

                entries.append(
                    Scenario(
                        name=task_name,
                        variant=variant_dir.name,
                        labels=[g["name"] for g in ground_truth_json],
                        db_schema=db_schema,
                        ground_truth=fromlist(Label, ground_truth_json),
                        code=variant_files[0].read_text(encoding="utf-8"),
                    )
                )

    # 2. write JSONL
    with open(output_file, "w", encoding="utf-8") as f:
        for entry in entries:
            f.write(json.dumps(asdict(entry), ensure_ascii=False) + "\n")

    print(f"Wrote {len(entries)} entries → {output_file}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()

    parser.add_argument("--tasks-dir", help="Path to the root tasks directory")
    parser.add_argument("--output-file", help="Destination .jsonl file")
    args = parser.parse_args()

    main(args.tasks_dir, args.output_file)
