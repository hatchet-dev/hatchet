# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""Stage 2: download failed-step logs once and extract failing tests + signatures.

Only failed jobs in gating workflows are parsed (those drive the ranking). Each
job's log is downloaded and parsed exactly once; results are cached per job id.
Expired logs (GitHub keeps them ~90 days) are cached as unparseable so they are
not retried forever.

    uv run parse_logs.py
"""

from __future__ import annotations

import sys
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from lib import cache, config, gh, parsers  # noqa: E402
from lib import signatures as sig  # noqa: E402

MAX_WORKERS = 8


def _failed_step(job: dict) -> str | None:
    for s in job.get("steps") or []:
        if s.get("conclusion") == "failure":
            return s.get("name")
    return None


def _parse_job(task: tuple[dict, int, dict]) -> dict:
    run, attempt, job = task
    wf_name = run["wf_name"]
    step = _failed_step(job)
    log = gh.api_text(f"repos/{config.REPO}/actions/jobs/{job['id']}/logs")
    tests_out: list[dict] = []
    parser_name = None
    if log is not None:
        items, parser_name = parsers.parse(wf_name, job["name"], log)
        for it in items:
            s = sig.signature(wf_name, job["name"], step, it["error_line"])
            tests_out.append({
                "test_id": it["test_id"],
                "error_line": sig.strip_ts(it["error_line"] or "")[:500],
                "signature": s,
                "sig_hash": sig.sig_hash(s),
            })
    job_sig = sig.signature(
        wf_name, job["name"], step, tests_out[0]["error_line"] if tests_out else step
    )
    return {
        "job_id": job["id"],
        "run_id": run["id"],
        "attempt": attempt,
        "workflow": wf_name,
        "job_name": job["name"],
        "failed_step": step,
        "parser": parser_name,
        "log_available": log is not None,
        "tests": tests_out,
        "job_signature": job_sig,
        "job_sig_hash": sig.sig_hash(job_sig),
    }


def main() -> int:
    tasks: list[tuple[dict, int, dict]] = []
    skipped = 0
    for run in cache.iter_runs():
        if not run.get("gating"):
            continue
        for attempt, jobs in (run.get("jobs_by_attempt") or {}).items():
            for job in jobs:
                if job.get("conclusion") != "failure":
                    continue
                if cache.has_job_failure(job["id"]):
                    skipped += 1
                    continue
                tasks.append((run, int(attempt), job))

    parsed = 0
    expired = 0
    with ThreadPoolExecutor(max_workers=MAX_WORKERS) as pool:
        for rec in pool.map(_parse_job, tasks):
            cache.put_job_failure(rec)
            parsed += 1
            if not rec["log_available"]:
                expired += 1
            if parsed % 50 == 0:
                print(f"  parsed {parsed}/{len(tasks)}", flush=True)

    print(f"parse_logs: {parsed} parsed, {skipped} cached, {expired} expired/no-log")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
