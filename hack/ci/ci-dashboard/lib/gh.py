"""Thin wrappers around the `gh` CLI. All GitHub access flows through here."""

from __future__ import annotations

import json
import subprocess
import time
from typing import Any, Callable

_TRANSIENT = ("502", "503", "timeout", "timed out", "rate limit", "abuse", "secondary",
              "connection reset", "eof")


class GhError(RuntimeError):
    pass


def _run(args: list[str], timeout: int = 180, retries: int = 3) -> str:
    last = ""
    for attempt in range(retries + 1):
        proc = subprocess.run(
            ["gh", *args],
            capture_output=True,
            text=True,
            timeout=timeout,
        )
        if proc.returncode == 0:
            return proc.stdout
        last = proc.stderr.strip()
        if attempt < retries and any(s in last.lower() for s in _TRANSIENT):
            time.sleep(2 * (attempt + 1))
            continue
        break
    raise GhError(f"gh {' '.join(args)} failed: {last}")


def api_json(path: str) -> Any:
    """GET a single JSON document."""
    return json.loads(_run(["api", path]))


def api_text(path: str, timeout: int = 180) -> str | None:
    """GET raw text (e.g. job logs). Returns None when the resource is gone (404/410)."""
    proc = subprocess.run(
        ["gh", "api", path],
        capture_output=True,
        text=True,
        timeout=timeout,
    )
    if proc.returncode != 0:
        stderr = proc.stderr.lower()
        if any(s in stderr for s in ("not found", "404", "410", "gone", "no logs")):
            return None
        raise GhError(f"gh api {path} failed ({proc.returncode}): {proc.stderr.strip()}")
    return proc.stdout


def paginate(
    path_base: str,
    key: str,
    stop_fn: Callable[[dict], bool] | None = None,
    per_page: int = 100,
    max_pages: int = 50,
) -> list[dict]:
    """Page through an object-wrapped list endpoint (e.g. {"workflow_runs": [...]}).

    When stop_fn returns True for an item, that item and the rest are dropped and
    pagination stops early. Items are assumed newest-first for early-stop to be valid.
    """
    sep = "&" if "?" in path_base else "?"
    results: list[dict] = []
    for page in range(1, max_pages + 1):
        obj = api_json(f"{path_base}{sep}per_page={per_page}&page={page}")
        items = obj.get(key, [])
        if not items:
            break
        stopped = False
        for item in items:
            if stop_fn is not None and stop_fn(item):
                stopped = True
                break
            results.append(item)
        if stopped or len(items) < per_page:
            break
    return results


def graphql(query: str, **fields: Any) -> Any:
    args = ["api", "graphql", "-f", f"query={query}"]
    for k, v in fields.items():
        args += ["-F" if not isinstance(v, str) else "-f", f"{k}={v}"]
    return json.loads(_run(args))


def pr_list(repo: str, label: str, state: str, limit: int, fields: list[str]) -> list[dict]:
    try:
        out = _run([
            "pr", "list",
            "--repo", repo,
            "--label", label,
            "--state", state,
            "--limit", str(limit),
            "--json", ",".join(fields),
        ])
    except GhError as exc:
        # Label may not exist yet (e.g. before backfill). Treat as no results.
        if "label" in str(exc).lower() or "could not" in str(exc).lower():
            return []
        raise
    return json.loads(out)
