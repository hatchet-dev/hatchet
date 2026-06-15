# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""Stage 4: collect recent CI-health wins (last merged / open `ci-health` PRs)
and merge them into out/analysis.json. Deterministic; returns empty lists if the
label does not exist yet.

    uv run wins.py
"""

from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from lib import cache, config, gh  # noqa: E402

FIELDS = ["number", "title", "url", "mergedAt", "createdAt", "author"]


def _slim(prs: list[dict]) -> list[dict]:
    out = []
    for pr in prs:
        out.append({
            "number": pr.get("number"),
            "title": pr.get("title"),
            "url": pr.get("url"),
            "merged_at": pr.get("mergedAt"),
            "created_at": pr.get("createdAt"),
            "author": (pr.get("author") or {}).get("login"),
        })
    return out


def main() -> int:
    analysis = cache.read_json(config.ANALYSIS_FILE, default={}) or {}
    merged = gh.pr_list(config.REPO, config.DASHBOARD_LABEL, "merged", 5, FIELDS)
    opened = gh.pr_list(config.REPO, config.DASHBOARD_LABEL, "open", 5, FIELDS)
    analysis["wins"] = {"merged": _slim(merged), "open": _slim(opened)}
    cache.write_json(config.ANALYSIS_FILE, analysis)
    print(f"wins: {len(merged)} merged, {len(opened)} open '{config.DASHBOARD_LABEL}' PRs")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
