# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""Stage 3: turn the cache into out/analysis.json. Pure compute, no network.

Computes, over gating workflows only:
  - per-job and per-test failure counts / rates,
  - flaky vs deterministic (re-run recovery, or intermittent),
  - PR-only vs main vs both scope,
  - daily failure trend,
  - overall PR/main pass rates,
  - the set of failure signatures that need classification.

    uv run aggregate.py
"""

from __future__ import annotations

import sys
from collections import Counter, defaultdict
from datetime import datetime, timedelta, timezone
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from lib import cache, config  # noqa: E402
from lib import signatures as sig  # noqa: E402


def _parse_ts(ts: str) -> datetime | None:
    if not ts:
        return None
    if ts.endswith("Z"):
        ts = ts[:-1] + "+00:00"
    try:
        return datetime.fromisoformat(ts)
    except ValueError:
        return None


def _in_table_window(run: dict, cutoff: datetime) -> bool:
    created = _parse_ts(run.get("created_at") or "")
    return created is not None and created >= cutoff


def _scope(run: dict) -> str:
    event = run.get("event")
    if event == "push" and run.get("head_branch") == "main":
        return "main"
    if event == "pull_request":
        return "pr"
    return "other"


def _label_scope(pr: int, main: int) -> str:
    if pr and main:
        return "both"
    if main:
        return "main"
    if pr:
        return "pr"
    return "none"


def _flaky_label(runs_total: int, fails: int, recovered: int) -> str:
    if recovered > 0:
        return "flaky"
    passes = runs_total - fails
    if fails > 0 and passes > 0:
        return "flaky"
    if fails > 0 and passes == 0:
        return "deterministic"
    return "n/a"


def main() -> int:
    runs = [r for r in cache.iter_runs() if r.get("gating")]
    runs_by_id = {r["id"]: r for r in runs}
    table_cutoff = datetime.now(timezone.utc) - timedelta(hours=config.TABLE_WINDOW_HOURS)

    # job-level accumulation, keyed by (workflow, stripped job name)
    jobs: dict[tuple, dict] = defaultdict(lambda: {
        "fails": 0, "recovered": 0, "pr_fails": 0, "main_fails": 0,
    })
    trend: dict[str, dict] = defaultdict(
        lambda: {"pr_runs": 0, "pr_fails": 0, "main_runs": 0, "main_fails": 0}
    )
    workflow_runs: Counter = Counter()  # denominator: runs per gating workflow

    # overall pass-rate totals (per gating workflow run)
    totals = {"pr_runs": 0, "pr_fail_runs": 0, "main_runs": 0, "main_fail_runs": 0}

    for run in runs:
        scope = _scope(run)
        date = (run.get("created_at") or "")[:10]
        in_table_window = _in_table_window(run, table_cutoff)
        if in_table_window:
            workflow_runs[run["wf_name"]] += 1

        # run-level totals + trend (independent of job detail, so clean successes
        # without fetched jobs still count toward denominators / pass rate)
        if scope in ("pr", "main"):
            totals[f"{scope}_runs"] += 1
            trend[date][f"{scope}_runs"] += 1
            if run.get("conclusion") == "failure":
                totals[f"{scope}_fail_runs"] += 1
                trend[date][f"{scope}_fails"] += 1

        if not in_table_window:
            continue

        by_attempt = run.get("jobs_by_attempt") or {}
        if not by_attempt:
            continue
        attempts = sorted(int(a) for a in by_attempt)
        final_n = attempts[-1]

        # gather per-job conclusions across attempts
        job_attempt_concl: dict[str, dict[int, str]] = defaultdict(dict)
        for a in attempts:
            for job in by_attempt[str(a)]:
                job_attempt_concl[sig.strip_matrix(job["name"])][a] = job.get("conclusion")

        for jname, concl in job_attempt_concl.items():
            rec = jobs[(run["wf_name"], jname)]
            final = concl.get(final_n)
            ever_failed = any(c == "failure" for c in concl.values())
            if final == "failure":
                rec["fails"] += 1
                if scope == "pr":
                    rec["pr_fails"] += 1
                elif scope == "main":
                    rec["main_fails"] += 1
            elif ever_failed and final == "success":
                rec["recovered"] += 1

    # representative signature per job (from parsed job-failures)
    job_sigs: dict[tuple, Counter] = defaultdict(Counter)
    job_sig_sample: dict[str, dict] = {}
    # test-level accumulation, keyed by (workflow, stripped job, test_id-or-sig)
    tests: dict[tuple, dict] = defaultdict(lambda: {
        "runs": set(), "sigs": Counter(), "samples": {},
        "test_id": None, "pr": set(), "main": set(),
    })

    for jf in cache.iter_job_failures():
        run = runs_by_id.get(jf["run_id"])
        if run is None or not _in_table_window(run, table_cutoff):
            continue
        scope = _scope(run)
        jkey = (jf["workflow"], sig.strip_matrix(jf["job_name"]))
        job_sigs[jkey][jf["job_sig_hash"]] += 1
        if jf["job_sig_hash"] not in job_sig_sample:
            job_sig_sample[jf["job_sig_hash"]] = {
                "signature": jf["job_signature"],
                "sample_error": (jf["tests"][0]["error_line"] if jf["tests"] else jf.get("failed_step")),
                "workflow": jf["workflow"],
                "job": sig.strip_matrix(jf["job_name"]),
                "step": jf.get("failed_step"),
                "test_id": None,
            }
        for t in jf["tests"]:
            tid = t["test_id"] or t["sig_hash"]
            tkey = (jf["workflow"], sig.strip_matrix(jf["job_name"]), tid)
            trec = tests[tkey]
            trec["test_id"] = t["test_id"]
            trec["runs"].add(jf["run_id"])
            trec["sigs"][t["sig_hash"]] += 1
            trec["samples"].setdefault(t["sig_hash"], {
                "signature": t["signature"],
                "sample_error": t["error_line"],
                "workflow": jf["workflow"],
                "job": sig.strip_matrix(jf["job_name"]),
                "step": jf.get("failed_step"),
                "test_id": t["test_id"],
            })
            if scope == "pr":
                trec["pr"].add(jf["run_id"])
            elif scope == "main":
                trec["main"].add(jf["run_id"])

    # build top jobs
    top_jobs = []
    for (wf, jname), rec in jobs.items():
        problems = rec["fails"] + rec["recovered"]
        if problems == 0:
            continue
        runs_total = workflow_runs.get(wf, rec["fails"]) or rec["fails"]
        top_sig = job_sigs[(wf, jname)].most_common(1)
        sig_hash = top_sig[0][0] if top_sig else None
        top_jobs.append({
            "workflow": wf,
            "job": jname,
            "fails": rec["fails"],
            "recovered": rec["recovered"],
            "runs_total": runs_total,
            "fail_rate": round(rec["fails"] / runs_total, 3) if runs_total else 0.0,
            "flaky": _flaky_label(runs_total, rec["fails"], rec["recovered"]),
            "scope": _label_scope(rec["pr_fails"], rec["main_fails"]),
            "sig_hash": sig_hash,
        })
    top_jobs.sort(key=lambda j: (j["fails"] + j["recovered"], j["fail_rate"]), reverse=True)
    top_jobs = top_jobs[: config.TOP_N]

    # build top tests
    top_tests = []
    for (wf, jname, tid), rec in tests.items():
        count = len(rec["runs"])
        if count == 0:
            continue
        jrec = jobs.get((wf, jname), {})
        runs_total = workflow_runs.get(wf, count) or count
        recovered = jrec.get("recovered", 0)
        top_sig = rec["sigs"].most_common(1)
        sig_hash = top_sig[0][0] if top_sig else None
        top_tests.append({
            "workflow": wf,
            "job": jname,
            "test_id": rec["test_id"] or "(unparsed)",
            "fails": count,
            "runs_total": runs_total,
            "fail_rate": round(count / runs_total, 3) if runs_total else 0.0,
            "flaky": _flaky_label(runs_total, count, recovered),
            "scope": _label_scope(len(rec["pr"]), len(rec["main"])),
            "sig_hash": sig_hash,
        })
    top_tests.sort(key=lambda t: (t["fails"], t["fail_rate"]), reverse=True)
    top_tests = top_tests[: config.TOP_N]

    # signatures needing classification = those referenced by top items
    wanted = {j["sig_hash"] for j in top_jobs if j["sig_hash"]}
    wanted |= {t["sig_hash"] for t in top_tests if t["sig_hash"]}
    signatures = {}
    all_samples = dict(job_sig_sample)
    for trec in tests.values():
        all_samples.update(trec["samples"])
    for h in wanted:
        if h in all_samples:
            signatures[h] = all_samples[h]

    def pass_rate(total: int, fails: int) -> float | None:
        return round(1 - fails / total, 3) if total else None

    # trend, sorted by date ascending, with per-day pass rates
    trend_list = [
        {
            "date": d,
            "pr_runs": v["pr_runs"],
            "pr_fails": v["pr_fails"],
            "main_runs": v["main_runs"],
            "main_fails": v["main_fails"],
            "pr_pass_rate": pass_rate(v["pr_runs"], v["pr_fails"]),
            "main_pass_rate": pass_rate(v["main_runs"], v["main_fails"]),
        }
        for d, v in sorted(trend.items())
    ]

    analysis = {
        "generated_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "window_days": config.WINDOW_DAYS,
        "table_window_hours": config.TABLE_WINDOW_HOURS,
        "since": cache.load_meta().get("since"),
        "totals": {
            **totals,
            "pr_pass_rate": pass_rate(totals["pr_runs"], totals["pr_fail_runs"]),
            "main_pass_rate": pass_rate(totals["main_runs"], totals["main_fail_runs"]),
        },
        "trend": trend_list,
        "top_jobs": top_jobs,
        "top_tests": top_tests,
        "signatures": signatures,
        "wins": cache.read_json(config.ANALYSIS_FILE, default={}).get("wins", {}) if config.ANALYSIS_FILE.exists() else {},
    }
    cache.write_json(config.ANALYSIS_FILE, analysis)
    print(
        f"aggregate: {len(top_jobs)} top jobs, {len(top_tests)} top tests, "
        f"{len(signatures)} signatures to classify"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
