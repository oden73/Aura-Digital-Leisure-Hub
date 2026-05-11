#!/usr/bin/env python3
"""Generate dependency-free visual reports from backend test artifacts."""

from __future__ import annotations

import argparse
import html
import re
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable


@dataclass(frozen=True)
class Status:
    name: str
    code: int | None


@dataclass(frozen=True)
class CoverageRow:
    name: str
    percent: float


@dataclass(frozen=True)
class Scenario:
    name: str
    metrics: dict[str, float]


@dataclass(frozen=True)
class RecommendationQuality:
    summary: dict[str, float]
    scenarios: list[Scenario]


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Build HTML and SVG visualizations from backend reports."
    )
    parser.add_argument(
        "--reports-dir",
        default="reports",
        type=Path,
        help="Directory with backend reports, relative paths are resolved from cwd.",
    )
    args = parser.parse_args()

    reports_dir = args.reports_dir
    reports_dir.mkdir(parents=True, exist_ok=True)

    test_report = reports_dir / "test-report.txt"
    quality_report = reports_dir / "recommendation_quality_report.md"

    test_text = read_text(test_report)
    quality = parse_recommendation_quality(read_text(quality_report))
    statuses = parse_statuses(test_text)
    go_coverage = parse_go_package_coverage(test_text)
    python_coverage = parse_python_coverage(test_text)
    totals = parse_coverage_totals(test_text, go_coverage, python_coverage)

    coverage_svg = render_coverage_svg(totals, go_coverage, python_coverage)
    quality_svg = render_quality_svg(quality)
    dashboard_html = render_dashboard_html(
        statuses=statuses,
        totals=totals,
        go_coverage=go_coverage,
        python_coverage=python_coverage,
        quality=quality,
        coverage_svg_name="coverage_overview.svg",
        quality_svg_name="recommendation_quality_overview.svg",
        source_reports=[
            path.name
            for path in (test_report, quality_report)
            if path.exists()
        ],
    )

    (reports_dir / "coverage_overview.svg").write_text(coverage_svg, encoding="utf-8")
    (reports_dir / "recommendation_quality_overview.svg").write_text(
        quality_svg, encoding="utf-8"
    )
    (reports_dir / "visual-report.html").write_text(dashboard_html, encoding="utf-8")

    print(f"visual reports written to {reports_dir}")
    return 0


def read_text(path: Path) -> str:
    if not path.exists():
        return ""
    return path.read_text(encoding="utf-8", errors="replace")


def parse_statuses(text: str) -> list[Status]:
    statuses = []
    for key, label in (("go_status", "Go core"), ("python_status", "Python AI engine")):
        match = re.search(rf"^{re.escape(key)}=(\d+)$", text, re.MULTILINE)
        statuses.append(Status(label, int(match.group(1)) if match else None))
    return statuses


def parse_go_package_coverage(text: str) -> list[CoverageRow]:
    rows: list[CoverageRow] = []
    for match in re.finditer(
        r"^ok\s+(\S+).*?coverage:\s+([0-9]+(?:\.[0-9]+)?)%",
        text,
        re.MULTILINE,
    ):
        rows.append(CoverageRow(short_go_package(match.group(1)), float(match.group(2))))
    return rows


def parse_python_coverage(text: str) -> list[CoverageRow]:
    rows: list[CoverageRow] = []
    in_table = False
    for line in text.splitlines():
        if line.startswith("Python app coverage summary:"):
            in_table = True
            continue
        if line.startswith("Name") and "Cover" in line:
            in_table = True
            continue
        if not in_table or not line.strip():
            continue
        if (
            line.startswith("module")
            or line.startswith("Name")
            or line.startswith("TOTAL")
            or line.startswith("---")
        ):
            continue
        match = re.match(r"^(\S+)\s+\d+\s+\d+\s+(\d+)%\b", line)
        if match:
            rows.append(CoverageRow(match.group(1), float(match.group(2))))
    return rows


def parse_coverage_totals(
    text: str,
    go_rows: list[CoverageRow],
    python_rows: list[CoverageRow],
) -> list[CoverageRow]:
    totals: list[CoverageRow] = []
    go_total = re.search(r"^total:\s+\(statements\)\s+([0-9]+(?:\.[0-9]+)?)%", text, re.MULTILINE)
    if go_total:
        totals.append(CoverageRow("Go total", float(go_total.group(1))))
    elif go_rows:
        totals.append(CoverageRow("Go total", average(row.percent for row in go_rows)))

    python_total = re.search(r"^TOTAL\s+\d+\s+\d+\s+(\d+)%$", text, re.MULTILINE)
    if python_total:
        totals.append(CoverageRow("Python total", float(python_total.group(1))))
    elif python_rows:
        totals.append(CoverageRow("Python total", average(row.percent for row in python_rows)))
    return totals


def parse_recommendation_quality(text: str) -> RecommendationQuality | None:
    if not text.strip():
        return None

    summary: dict[str, float] = {}
    scenarios: list[Scenario] = []
    current_name: str | None = None
    current_metrics: dict[str, float] = {}

    for line in text.splitlines():
        if line.startswith("### "):
            if current_name is not None:
                scenarios.append(Scenario(current_name, current_metrics))
            current_name = line.removeprefix("### ").strip()
            current_metrics = {}
            continue

        match = re.match(r"^-\s+([a-z_]+)(?:@\d+)?:\s+([0-9]+(?:\.[0-9]+)?)$", line)
        if not match:
            continue
        name, value = normalize_metric(match.group(1)), float(match.group(2))
        if current_name is None:
            summary[name] = value
        else:
            current_metrics[name] = value

    if current_name is not None:
        scenarios.append(Scenario(current_name, current_metrics))

    return RecommendationQuality(summary, scenarios)


def render_dashboard_html(
    *,
    statuses: list[Status],
    totals: list[CoverageRow],
    go_coverage: list[CoverageRow],
    python_coverage: list[CoverageRow],
    quality: RecommendationQuality | None,
    coverage_svg_name: str,
    quality_svg_name: str,
    source_reports: list[str],
) -> str:
    status_cards = "\n".join(render_status_card(status) for status in statuses)
    total_cards = "\n".join(render_metric_card(row.name, f"{row.percent:.1f}%") for row in totals)
    quality_cards = ""
    if quality:
        quality_cards = "\n".join(
            render_metric_card(label, f"{quality.summary[key]:.3f}")
            for key, label in (
                ("precision", "Precision"),
                ("recall", "Recall"),
                ("ndcg", "NDCG"),
                ("mrr", "MRR"),
                ("hit_rate", "Hit rate"),
                ("catalog_coverage", "Catalog coverage"),
                ("genre_diversity", "Genre diversity"),
            )
            if key in quality.summary
        )

    source_links = "\n".join(
        f'<li><a href="{escape_attr(name)}">{escape(name)}</a></li>'
        for name in source_reports
    )

    return f"""<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Backend Visual Test Report</title>
  <style>
    :root {{
      color-scheme: light;
      --bg: #f6f8fb;
      --card: #ffffff;
      --ink: #172033;
      --muted: #637083;
      --border: #dfe6ef;
      --ok: #16845b;
      --fail: #b42318;
      --accent: #4169e1;
    }}
    body {{
      margin: 0;
      background: var(--bg);
      color: var(--ink);
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }}
    main {{
      box-sizing: border-box;
      max-width: 1180px;
      margin: 0 auto;
      padding: 32px 20px 48px;
    }}
    h1, h2 {{
      margin: 0;
      letter-spacing: -0.02em;
    }}
    h1 {{
      font-size: clamp(28px, 4vw, 44px);
    }}
    h2 {{
      font-size: 22px;
      margin-bottom: 16px;
    }}
    p {{
      color: var(--muted);
      line-height: 1.55;
    }}
    section {{
      margin-top: 28px;
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 18px;
      padding: 22px;
      box-shadow: 0 12px 32px rgba(23, 32, 51, 0.06);
    }}
    .grid {{
      display: grid;
      gap: 14px;
      grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
    }}
    .card {{
      border: 1px solid var(--border);
      border-radius: 14px;
      padding: 16px;
      background: linear-gradient(180deg, #fff, #f9fbff);
    }}
    .label {{
      color: var(--muted);
      font-size: 13px;
      text-transform: uppercase;
      letter-spacing: 0.06em;
    }}
    .value {{
      display: block;
      margin-top: 7px;
      font-size: 28px;
      font-weight: 750;
    }}
    .ok {{ color: var(--ok); }}
    .fail {{ color: var(--fail); }}
    .visual {{
      display: block;
      width: 100%;
      max-width: 1040px;
      margin: 8px auto 0;
      border: 1px solid var(--border);
      border-radius: 16px;
      background: #fff;
    }}
    .bars {{
      display: grid;
      gap: 10px;
    }}
    .bar-row {{
      display: grid;
      grid-template-columns: minmax(220px, 1.2fr) minmax(180px, 3fr) 64px;
      align-items: center;
      gap: 12px;
      font-size: 14px;
    }}
    .bar-row > span:first-child {{
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }}
    .bar-track {{
      height: 12px;
      border-radius: 999px;
      background: #e8eef7;
      overflow: hidden;
    }}
    .bar-fill {{
      height: 100%;
      border-radius: inherit;
      background: linear-gradient(90deg, #5b7cfa, #22a06b);
    }}
    ul {{
      margin: 0;
      padding-left: 20px;
      color: var(--muted);
    }}
    a {{ color: var(--accent); }}
  </style>
</head>
<body>
  <main>
    <header>
      <h1>Backend Visual Test Report</h1>
      <p>Generated from the same artifacts that are written to <code>backend/reports</code>.</p>
    </header>

    <section>
      <h2>Run Status</h2>
      <div class="grid">
        {status_cards}
      </div>
    </section>

    <section>
      <h2>Coverage Totals</h2>
      <div class="grid">
        {total_cards or render_empty_card("Coverage totals not found")}
      </div>
      <img class="visual" src="{escape_attr(coverage_svg_name)}" alt="Coverage overview chart">
    </section>

    <section>
      <h2>Recommendation Quality</h2>
      <div class="grid">
        {quality_cards or render_empty_card("Recommendation report not found")}
      </div>
      <img class="visual" src="{escape_attr(quality_svg_name)}" alt="Recommendation quality chart">
    </section>

    <section>
      <h2>Go Package Coverage</h2>
      {render_html_bars(go_coverage[:12]) or render_empty_card("Go package coverage not found")}
    </section>

    <section>
      <h2>Python Module Coverage</h2>
      {render_html_bars(python_coverage[:12]) or render_empty_card("Python module coverage not found")}
    </section>

    <section>
      <h2>Source Reports</h2>
      <ul>
        {source_links or "<li>No source reports found.</li>"}
      </ul>
    </section>
  </main>
</body>
</html>
"""


def render_coverage_svg(
    totals: list[CoverageRow],
    go_coverage: list[CoverageRow],
    python_coverage: list[CoverageRow],
) -> str:
    rows = totals + lowest_rows("Go low coverage", go_coverage, 6) + lowest_rows(
        "Python low coverage", python_coverage, 6
    )
    return render_bar_svg(
        title="Coverage Overview",
        rows=rows,
        width=1120,
        row_height=32,
        max_value=100.0,
        value_suffix="%",
    )


def render_quality_svg(quality: RecommendationQuality | None) -> str:
    if not quality:
        return render_bar_svg(
            title="Recommendation Quality",
            rows=[CoverageRow("No recommendation report", 0)],
            width=1120,
            row_height=32,
            max_value=1.0,
            value_suffix="",
        )

    rows = [
        CoverageRow(label, quality.summary[key])
        for key, label in (
            ("precision", "Precision"),
            ("recall", "Recall"),
            ("ndcg", "NDCG"),
            ("mrr", "MRR"),
            ("hit_rate", "Hit rate"),
            ("catalog_coverage", "Catalog coverage"),
            ("genre_diversity", "Genre diversity"),
        )
        if key in quality.summary
    ]
    rows.extend(
        CoverageRow(f"{scenario.name}: precision", scenario.metrics["precision"])
        for scenario in quality.scenarios
        if "precision" in scenario.metrics
    )
    return render_bar_svg(
        title="Recommendation Quality",
        rows=rows,
        width=1120,
        row_height=32,
        max_value=1.0,
        value_suffix="",
    )


def render_bar_svg(
    *,
    title: str,
    rows: list[CoverageRow],
    width: int,
    row_height: int,
    max_value: float,
    value_suffix: str,
) -> str:
    left = 380
    right = 112
    top = 70
    bar_width = width - left - right
    height = top + len(rows) * row_height + 28
    parts = [
        svg_header(width, height),
        f'<rect width="100%" height="100%" rx="18" fill="#ffffff"/>',
        f'<text x="28" y="40" font-size="24" font-weight="700" fill="#172033">{escape(title)}</text>',
    ]
    for index, row in enumerate(rows):
        y = top + index * row_height
        value = max(0.0, min(row.percent, max_value))
        fill_width = 0 if max_value == 0 else bar_width * value / max_value
        color = color_for_percent(value, max_value)
        value_text = f"{row.percent:.1f}{value_suffix}" if value_suffix else f"{row.percent:.3f}"
        label = truncate_middle(row.name, 46)
        title_node = "" if label == row.name else f"<title>{escape(row.name)}</title>"
        parts.extend(
            [
                f'<text x="28" y="{y + 18}" font-size="13" fill="#344054">{title_node}{escape(label)}</text>',
                f'<rect x="{left}" y="{y + 4}" width="{bar_width}" height="16" rx="8" fill="#e8eef7"/>',
                f'<rect x="{left}" y="{y + 4}" width="{fill_width:.1f}" height="16" rx="8" fill="{color}"/>',
                f'<text x="{left + bar_width + 16}" y="{y + 18}" font-size="13" font-weight="700" fill="#172033">{escape(value_text)}</text>',
            ]
        )
    parts.append("</svg>")
    return "\n".join(parts)


def render_status_card(status: Status) -> str:
    if status.code is None:
        value = "missing"
        class_name = "fail"
    elif status.code == 0:
        value = "passed"
        class_name = "ok"
    else:
        value = f"failed ({status.code})"
        class_name = "fail"
    return (
        '<div class="card">'
        f'<span class="label">{escape(status.name)}</span>'
        f'<span class="value {class_name}">{escape(value)}</span>'
        "</div>"
    )


def render_metric_card(label: str, value: str) -> str:
    return (
        '<div class="card">'
        f'<span class="label">{escape(label)}</span>'
        f'<span class="value">{escape(value)}</span>'
        "</div>"
    )


def render_empty_card(message: str) -> str:
    return f'<div class="card"><span class="label">{escape(message)}</span></div>'


def render_html_bars(rows: list[CoverageRow]) -> str:
    if not rows:
        return ""
    bars = []
    for row in rows:
        width = max(0.0, min(row.percent, 100.0))
        bars.append(
            '<div class="bar-row">'
            f'<span title="{escape_attr(row.name)}">{escape(row.name)}</span>'
            '<span class="bar-track">'
            f'<span class="bar-fill" style="display:block;width:{width:.1f}%"></span>'
            "</span>"
            f"<strong>{row.percent:.1f}%</strong>"
            "</div>"
        )
    return f'<div class="bars">{"".join(bars)}</div>'


def lowest_rows(prefix: str, rows: list[CoverageRow], limit: int) -> list[CoverageRow]:
    return [
        CoverageRow(f"{prefix}: {row.name}", row.percent)
        for row in sorted(rows, key=lambda row: row.percent)[:limit]
    ]


def short_go_package(package: str) -> str:
    marker = "/core-go/"
    if marker in package:
        return package.split(marker, 1)[1]
    return package.rsplit("/", 1)[-1]


def normalize_metric(metric: str) -> str:
    if metric.startswith("precision"):
        return "precision"
    if metric.startswith("recall"):
        return "recall"
    if metric.startswith("ndcg"):
        return "ndcg"
    if metric.startswith("mrr"):
        return "mrr"
    if metric.startswith("hit_rate"):
        return "hit_rate"
    if metric.startswith("catalog_coverage"):
        return "catalog_coverage"
    if metric.startswith("genre_diversity"):
        return "genre_diversity"
    return metric


def average(values: Iterable[float]) -> float:
    collected = list(values)
    if not collected:
        return 0.0
    return sum(collected) / len(collected)


def truncate_middle(value: str, limit: int) -> str:
    if len(value) <= limit:
        return value
    if limit <= 3:
        return "." * limit
    keep = limit - 3
    left = keep // 2
    right = keep - left
    return f"{value[:left]}...{value[-right:]}"


def color_for_percent(value: float, max_value: float) -> str:
    ratio = 0.0 if max_value == 0 else value / max_value
    if ratio >= 0.85:
        return "#22a06b"
    if ratio >= 0.6:
        return "#f5a524"
    return "#d92d20"


def svg_header(width: int, height: int) -> str:
    return (
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" '
        f'viewBox="0 0 {width} {height}" role="img">'
    )


def escape(value: str) -> str:
    return html.escape(value, quote=False)


def escape_attr(value: str) -> str:
    return html.escape(value, quote=True)


if __name__ == "__main__":
    raise SystemExit(main())
