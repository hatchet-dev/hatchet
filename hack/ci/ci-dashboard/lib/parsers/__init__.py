"""Pick a log parser by workflow / job name and extract failing tests.

The generic parser is always used as a fallback so every failure gets at least a
signature, even when the ecosystem-specific parser finds nothing."""

from __future__ import annotations

from lib.parsers import cypress, generic, go, jest, pytest, rspec


def _pick(workflow: str, job_name: str):
    wf = (workflow or "").lower()
    job = (job_name or "").lower()

    if wf == "python" or "pytest" in job or "python" in job:
        return pytest
    if wf == "ruby" or "rspec" in job:
        return rspec
    if "cypress" in wf or "cypress" in job:
        return cypress
    if wf == "typescript" or "jest" in job or "vitest" in job or wf.startswith("frontend"):
        return jest
    if wf in ("go", "test", "build", "e2e-image", "cli-e2e-tests") or "e2e" in job or "load" in job:
        return go
    return generic


def parse(workflow: str, job_name: str, log: str) -> tuple[list[dict], str]:
    """Return (failing tests, parser-name). Falls back to generic when empty."""
    mod = _pick(workflow, job_name)
    items = mod.parse(log)
    parser_name = mod.__name__.rsplit(".", 1)[-1]
    if not items:
        items = generic.parse(log)
        if items:
            parser_name = "generic"
    return items, parser_name
