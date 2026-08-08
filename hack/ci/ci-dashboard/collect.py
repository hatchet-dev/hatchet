# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""Stage 1: incremental fetch of GitHub Actions runs into the local cache.

Run-level metadata is collected for every active workflow. Per-attempt job
details are fetched only for gating workflows (the ones that feed the ranking).
Completed runs are immutable, so a run cached at its current attempt is skipped;
only new runs / new attempts trigger network calls.

    uv run collect.py
"""

from __future__ import annotations

import sys
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timedelta, timezone
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from lib import cache, config, gh  # noqa: E402

MAX_WORKERS = 8  # parallelism for per-attempt job fetches
LIST_WORKERS = 3  # parallelism for per-workflow run listing (the steady-state cost)


def _since() -> str:
    return (datetime.now(timezone.utc) - timedelta(days=config.WINDOW_DAYS)).strftime("%Y-%m-%dT%H:%M:%SZ")


def _list_workflow_runs(wf: dict, since: str) -> tuple[dict, list[dict]]:
    runs = gh.paginate(
        f"repos/{config.REPO}/actions/workflows/{wf['id']}/runs",
        key="workflow_runs",
        stop_fn=lambda r: (r.get("created_at") or "") < since,
    )
    return wf, runs


def _fetch_attempt_jobs(run_id: int, attempt: int) -> list[dict]:
    raw = gh.paginate(
        f"repos/{config.REPO}/actions/runs/{run_id}/attempts/{attempt}/jobs",
        key="jobs",
    )
    jobs = []
    for j in raw:
        jobs.append({
            "id": j["id"],
            "name": j["name"],
            "status": j["status"],
            "conclusion": j["conclusion"],
            "started_at": j.get("started_at"),
            "completed_at": j.get("completed_at"),
            "steps": [
                {"name": s["name"], "number": s["number"], "conclusion": s["conclusion"]}
                for s in (j.get("steps") or [])
            ],
        })
    return jobs


def main() -> int:
    cache.ensure_dirs()
    since = _since()

    workflows = gh.api_json(f"repos/{config.REPO}/actions/workflows?per_page=100")["workflows"]
    active = [w for w in workflows if w.get("state") == "active"]
    print(f"collect: {len(active)} active workflows, window since {since}", flush=True)

    # Phase 1: list each workflow's runs in parallel. Pagination within a workflow
    # is sequential, but the 34 workflows are listed concurrently.
    listed: list[tuple[dict, list[dict]]] = []
    with ThreadPoolExecutor(max_workers=LIST_WORKERS) as lister:
        for wf, runs in lister.map(lambda w: _list_workflow_runs(w, since), active):
            listed.append((wf, runs))
            print(f"  listed {wf['name']}: {len(runs)} runs", flush=True)

    new_runs = 0
    new_attempts = 0
    api_run_calls = 0

    # Phase 2: build records + fetch job detail (failed/re-run gating runs only).
    with ThreadPoolExecutor(max_workers=MAX_WORKERS) as pool:
        for wf, runs in listed:
            wf_name = wf["name"]
            gating = wf_name in config.GATING_WORKFLOWS

            # First pass: build records and decide which (run, attempt) job pages to fetch.
            pending: list[tuple[dict, dict, list[int], bool]] = []
            considered = 0
            for r in runs:
                if r.get("status") != "completed":
                    continue  # skip in-progress; picked up next time
                considered += 1
                attempt = int(r.get("run_attempt") or 1)
                # Job/attempt detail is only needed to rank failing jobs/tests and to
                # detect re-run recovery. Clean single-attempt successes need none.
                need_jobs = gating and (r.get("conclusion") != "success" or attempt > 1)
                cached = cache.get_run(r["id"])
                if cached and int(cached.get("run_attempt", 0)) >= attempt and (
                    not need_jobs or cached.get("jobs_by_attempt")
                ):
                    continue

                record = cached or {}
                record.update({
                    "id": r["id"],
                    "wf_name": wf_name,
                    "workflow_id": r.get("workflow_id"),
                    "event": r.get("event"),
                    "head_branch": r.get("head_branch"),
                    "status": r.get("status"),
                    "conclusion": r.get("conclusion"),
                    "run_attempt": attempt,
                    "created_at": r.get("created_at"),
                    "updated_at": r.get("updated_at"),
                    "html_url": r.get("html_url"),
                    "gating": gating,
                })
                jobs_by_attempt = dict(record.get("jobs_by_attempt") or {})
                needed = (
                    [n for n in range(1, attempt + 1) if str(n) not in jobs_by_attempt]
                    if need_jobs else []
                )
                pending.append((record, jobs_by_attempt, needed, bool(cached)))

            # Second pass: fetch all needed job pages in parallel, then write.
            futures = {
                (rec["id"], n): pool.submit(_fetch_attempt_jobs, rec["id"], n)
                for rec, _jba, needed, _c in pending
                for n in needed
            }
            for rec, jba, needed, was_cached in pending:
                for n in needed:
                    jba[str(n)] = futures[(rec["id"], n)].result()
                    api_run_calls += 1
                rec["jobs_by_attempt"] = jba
                cache.put_run(rec)
                if was_cached:
                    new_attempts += 1
                else:
                    new_runs += 1
            print(f"  {wf_name}: {considered} runs, {len(pending)} written", flush=True)

    cache.save_meta({
        "last_collect": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "window_days": config.WINDOW_DAYS,
        "since": since,
    })
    print(f"collect: {new_runs} new runs, {new_attempts} updated, {api_run_calls} job fetches")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
