# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""Stage 5: render out/analysis.json (+ classifications) into out/issue.md.

Deterministic. Produces the full issue body: header + totals, a single
normalized Mermaid PR pass-rate trend chart, top-failing-jobs and
top-failing-tests tables, and the wins section.

    uv run render.py
"""

from __future__ import annotations

import sys
from datetime import datetime, timezone
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from lib import cache, config  # noqa: E402

FLAKY_BADGE = {"flaky": "flaky", "deterministic": "deterministic", "n/a": "-"}
SCOPE_LABEL = {"both": "main + PR", "main": "main", "pr": "PR", "none": "-"}


def _pct(x: float | None) -> str:
    return f"{round(x * 100)}%" if x is not None else "n/a"


def _scope(s: str) -> str:
    return SCOPE_LABEL.get(s, s)


def _cause(sig_hash: str | None, classifications: dict) -> str:
    if not sig_hash or sig_hash not in classifications:
        return "_unclassified_"
    c = classifications[sig_hash]
    reason = (c.get("reason") or "").replace("|", "\\|")
    return f"**{c.get('category')}** — {reason}"


_MONTHS = {
    "01": "Jan", "02": "Feb", "03": "Mar", "04": "Apr", "05": "May", "06": "Jun",
    "07": "Jul", "08": "Aug", "09": "Sep", "10": "Oct", "11": "Nov", "12": "Dec",
}


def _fmt_day(date: str) -> str:
    """2026-06-01 -> 'Jun 01'."""
    return f"{_MONTHS.get(date[5:7], date[5:7])} {date[8:10]}"


def _ffill_pct(trend: list[dict], key: str) -> list[int]:
    """Per-day pass-rate series as 0-100 ints; carry the last known value across
    days a scope had no runs (so the line stays continuous instead of diving to 0).
    """
    first = next((t[key] for t in trend if t.get(key) is not None), None)
    last = round(first * 100) if first is not None else 0
    out = []
    for t in trend:
        v = t.get(key)
        if v is not None:
            last = round(v * 100)
        out.append(last)
    return out


def _mermaid_trend(trend: list[dict]) -> str:
    """Two-line normalized chart: CI (PR) and main gating-CI pass rate (%) per day.

    Kept to the most broadly-supported xychart-beta syntax so it renders across
    Mermaid versions (GitHub + Cursor preview): a categorical (band) x-axis with
    compact, unquoted day-of-month labels (consecutive days never collide), named
    line series, and no %%{init}%% / frontmatter / x-axis-title extensions (those
    triggered blank or syntax-error renders in some Mermaid builds). The date range
    and line identity live in the markdown caption, which always renders.
    """
    if not trend:
        return "_No gating-CI runs recorded in the window._"
    days = ", ".join(str(int(t["date"][8:10])) for t in trend)
    span = f"{_fmt_day(trend[0]['date'])} \u2192 {_fmt_day(trend[-1]['date'])}"
    pr = ", ".join(str(v) for v in _ffill_pct(trend, "pr_pass_rate"))
    main = ", ".join(str(v) for v in _ffill_pct(trend, "main_pass_rate"))
    return (
        "```mermaid\n"
        "xychart-beta\n"
        '  title "Gating-CI pass rate (%) per day"\n'
        f"  x-axis [{days}]\n"
        '  y-axis "pass rate %" 0 --> 100\n'
        f'  line "CI" [{pr}]\n'
        f'  line "main" [{main}]\n'
        "```\n\n"
        f"_X-axis = day of month ({span}). Two lines: **CI** (PR gating-CI runs, "
        "generally the upper line) and **main** (post-merge main runs, lower). "
        "Y-axis = % of that day's gating-CI runs that passed._"
    )


def _jobs_table(jobs: list[dict], classifications: dict) -> str:
    rows = ["| # | job | workflow | fails | recovered | runs | fail rate | flaky? | scope | cause |",
            "| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |"]
    for i, j in enumerate(jobs, 1):
        rows.append(
            f"| {i} | `{j['job']}` | {j['workflow']} | {j['fails']} | {j['recovered']} | "
            f"{j['runs_total']} | {_pct(j['fail_rate'])} | {FLAKY_BADGE.get(j['flaky'], j['flaky'])} | "
            f"{_scope(j['scope'])} | {_cause(j['sig_hash'], classifications)} |"
        )
    return "\n".join(rows) if jobs else "_No failing jobs._"


def _tests_table(tests: list[dict], classifications: dict) -> str:
    rows = ["| # | test | job | fails | runs | fail rate | flaky? | scope | cause |",
            "| --- | --- | --- | --- | --- | --- | --- | --- | --- |"]
    for i, t in enumerate(tests, 1):
        tid = (t["test_id"] or "(unparsed)").replace("|", "\\|")
        rows.append(
            f"| {i} | `{tid}` | `{t['job']}` | {t['fails']} | {t['runs_total']} | "
            f"{_pct(t['fail_rate'])} | {FLAKY_BADGE.get(t['flaky'], t['flaky'])} | "
            f"{_scope(t['scope'])} | {_cause(t['sig_hash'], classifications)} |"
        )
    return "\n".join(rows) if tests else "_No failing tests parsed._"


def _wins(wins: dict) -> str:
    def fmt(prs, when_key, when_label):
        if not prs:
            return f"_No {when_label} `{config.DASHBOARD_LABEL}` PRs yet._"
        # Bare PR URLs, one per line: GitHub auto-renders the title/reference.
        return "\n".join(f"- {pr['url']}" for pr in prs)

    merged = wins.get("merged", [])
    opened = wins.get("open", [])
    return (
        "**Recently merged**\n\n"
        + fmt(merged, "merged_at", "merged")
        + "\n\n**Open**\n\n"
        + fmt(opened, "created_at", "open")
    )


def main() -> int:
    analysis = cache.read_json(config.ANALYSIS_FILE)
    if analysis is None:
        print("render: no analysis.json; run aggregate.py first", file=sys.stderr)
        return 1
    classifications = cache.load_classifications()
    totals = analysis.get("totals", {})
    generated = analysis.get("generated_at", datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"))
    window_days = analysis.get("window_days", config.WINDOW_DAYS)
    table_hours = analysis.get("table_window_hours", config.TABLE_WINDOW_HOURS)

    parts = [
        f"# {config.DASHBOARD_TITLE}",
        "",
        f"_Window: last {window_days} days (trend + pass rate) · "
        f"tables: last {table_hours}h · "
        f"updated {generated} · auto-generated, do not edit by hand._",
        "",
        f"**Gating-CI pass rate** — PR: {_pct(totals.get('pr_pass_rate'))} "
        f"({totals.get('pr_runs', 0) - totals.get('pr_fail_runs', 0)}/{totals.get('pr_runs', 0)}) · "
        f"main: {_pct(totals.get('main_pass_rate'))} "
        f"({totals.get('main_runs', 0) - totals.get('main_fail_runs', 0)}/{totals.get('main_runs', 0)})",
        "",
        "## Gating-CI pass-rate trend",
        "",
        _mermaid_trend(analysis.get("trend", [])),
        "",
        f"## Top {config.TOP_N} failing jobs (last {table_hours}h)",
        "",
        _jobs_table(analysis.get("top_jobs", []), classifications),
        "",
        f"## Top {config.TOP_N} failing tests (last {table_hours}h)",
        "",
        _tests_table(analysis.get("top_tests", []), classifications),
        "",
        f"## Recent CI-health wins (`{config.DASHBOARD_LABEL}`)",
        "",
        _wins(analysis.get("wins", {})),
        "",
        "---",
        f"_Trend and pass-rate totals cover the last {window_days} days; "
        f"job/test tables cover the last {table_hours}h._ "
        "**fails** = gating runs where the job/test failed · "
        "**recovered** = failed on a first attempt but passed on re-run (a flakiness signal) · "
        "**runs** = total gating runs of that workflow · "
        "**fail rate** = fails ÷ runs · "
        "**flaky** = recovered on re-run or intermittent across runs; "
        "**deterministic** = fails every time it runs · "
        "**scope** = whether failures were seen on PR, main, or main + PR.",
    ]
    body = "\n".join(parts) + "\n"
    config.ISSUE_FILE.parent.mkdir(parents=True, exist_ok=True)
    config.ISSUE_FILE.write_text(body)
    print(f"render: wrote {config.ISSUE_FILE} ({len(body)} bytes)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
