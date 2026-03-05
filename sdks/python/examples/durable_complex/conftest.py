from __future__ import annotations

from typing import Any


def get_task_output(result: dict[str, Any], *preferred_keys: str) -> dict[str, Any]:
    """
    Extract task output from workflow result.

    Result keys may be task_name or workflowname:taskname. Falls back to
    first dict value when preferred keys are not found.
    """
    for key in preferred_keys:
        out = result.get(key)
        if out is not None and isinstance(out, dict):
            return dict(out)
    fallback: dict[str, Any] = next(
        (v for v in result.values() if isinstance(v, dict)), {}
    )
    return fallback
