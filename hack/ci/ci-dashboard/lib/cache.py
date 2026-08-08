"""Local on-disk cache. Completed runs/jobs/logs are immutable, so anything cached
is never re-fetched; only the delta since the last run is pulled. `rm -rf .cache/`
fully regenerates state."""

from __future__ import annotations

import json
import os
from pathlib import Path
from typing import Any, Iterator

from lib import config


def ensure_dirs() -> None:
    for d in (config.RUNS_DIR, config.JOB_FAILURES_DIR, config.OUT_DIR):
        d.mkdir(parents=True, exist_ok=True)


def read_json(path: Path, default: Any = None) -> Any:
    if not path.exists():
        return default
    with path.open() as f:
        return json.load(f)


def write_json(path: Path, data: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp = path.with_suffix(path.suffix + ".tmp")
    with tmp.open("w") as f:
        json.dump(data, f, indent=2, sort_keys=True)
    os.replace(tmp, path)


# --- runs ---------------------------------------------------------------

def run_path(run_id: int | str) -> Path:
    return config.RUNS_DIR / f"{run_id}.json"


def get_run(run_id: int | str) -> dict | None:
    return read_json(run_path(run_id))


def put_run(run: dict) -> None:
    write_json(run_path(run["id"]), run)


def iter_runs() -> Iterator[dict]:
    for p in sorted(config.RUNS_DIR.glob("*.json")):
        run = read_json(p)
        if run is not None:
            yield run


# --- job failures (test-level parse results) ----------------------------

def job_failure_path(job_id: int | str) -> Path:
    return config.JOB_FAILURES_DIR / f"{job_id}.json"


def has_job_failure(job_id: int | str) -> bool:
    return job_failure_path(job_id).exists()


def put_job_failure(record: dict) -> None:
    write_json(job_failure_path(record["job_id"]), record)


def iter_job_failures() -> Iterator[dict]:
    for p in sorted(config.JOB_FAILURES_DIR.glob("*.json")):
        rec = read_json(p)
        if rec is not None:
            yield rec


# --- classifications (signature -> cause) -------------------------------

def load_classifications() -> dict:
    return read_json(config.CLASSIFICATIONS_FILE, default={}) or {}


def save_classifications(data: dict) -> None:
    write_json(config.CLASSIFICATIONS_FILE, data)


# --- meta ---------------------------------------------------------------

def load_meta() -> dict:
    return read_json(config.META_FILE, default={}) or {}


def save_meta(data: dict) -> None:
    write_json(config.META_FILE, data)
