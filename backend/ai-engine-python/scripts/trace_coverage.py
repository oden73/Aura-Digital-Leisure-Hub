from __future__ import annotations

import argparse
import site
import sys
import trace
import unittest
from pathlib import Path


def module_name(root: Path, path: Path) -> str:
    rel = path.relative_to(root).with_suffix("")
    return ".".join(rel.parts)


def format_ranges(lines: list[int]) -> str:
    if not lines:
        return "-"
    ranges: list[str] = []
    start = prev = lines[0]
    for line in lines[1:]:
        if line == prev + 1:
            prev = line
            continue
        ranges.append(str(start) if start == prev else f"{start}-{prev}")
        start = prev = line
    ranges.append(str(start) if start == prev else f"{start}-{prev}")
    return ", ".join(ranges)


def print_app_summary(results: trace.CoverageResults, app_root: Path) -> None:
    counts_by_file: dict[Path, set[int]] = {}
    for (filename, lineno), count in results.counts.items():
        path = Path(filename).resolve()
        if count > 0 and path.is_relative_to(app_root):
            counts_by_file.setdefault(path, set()).add(lineno)

    rows: list[tuple[str, int, int, int, str]] = []
    total_statements = 0
    total_covered = 0

    for path in sorted(app_root.rglob("*.py")):
        executable = set(trace._find_executable_linenos(str(path)))
        if not executable:
            continue
        covered = counts_by_file.get(path.resolve(), set()) & executable
        missing = sorted(executable - covered)
        statements = len(executable)
        covered_count = len(covered)
        percent = int(round(covered_count / statements * 100))

        total_statements += statements
        total_covered += covered_count
        rows.append((
            module_name(app_root.parent, path),
            statements,
            covered_count,
            percent,
            format_ranges(missing),
        ))

    print("Python app coverage summary:")
    print("module                                          stmts  cover  cov%  missing")
    for module, statements, covered_count, percent, missing in rows:
        print(f"{module:<47} {statements:>5} {covered_count:>6} {percent:>4}%  {missing}")

    total_percent = int(round(total_covered / total_statements * 100)) if total_statements else 100
    print(f"TOTAL{''.ljust(42)} {total_statements:>5} {total_covered:>6} {total_percent:>4}%")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--coverdir", default="reports/pytrace")
    parser.add_argument("--start-dir", default="tests")
    args = parser.parse_args()

    root = Path.cwd().resolve()
    app_root = root / "app"
    sys.path.insert(0, str(root))
    coverdir = Path(args.coverdir)
    coverdir.mkdir(parents=True, exist_ok=True)

    ignoredirs = {sys.prefix, sys.exec_prefix}
    for path in site.getsitepackages():
        ignoredirs.add(path)
    user_site = site.getusersitepackages()
    if user_site:
        ignoredirs.add(user_site)
    ignoredirs.add(str(Path.home() / ".local"))

    tracer = trace.Trace(
        count=True,
        trace=False,
        ignoredirs=[str(Path(p).resolve()) for p in ignoredirs],
        ignoremods=[
            "unittest",
            "trace",
            "argparse",
            "site",
            "pydantic",
            "pydantic.main",
            "test_adapters",
            "test_config_and_main",
            "test_content_similarity",
            "test_embedding_provider",
            "test_llm_reasoning",
            "test_routes",
            "test_schemas_and_cold_start",
        ],
    )

    def run_tests() -> unittest.TestResult:
        suite = unittest.defaultTestLoader.discover(args.start_dir)
        runner = unittest.TextTestRunner(verbosity=1)
        return runner.run(suite)

    result = tracer.runfunc(run_tests)
    print()
    print_app_summary(tracer.results(), app_root)

    return 0 if result.wasSuccessful() else 1


if __name__ == "__main__":
    raise SystemExit(main())
