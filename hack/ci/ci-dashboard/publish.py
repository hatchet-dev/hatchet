# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""Stage 6: publish out/issue.md to the pinned dashboard issue.

Finds the dashboard issue by a hidden marker in its body, creates it if missing,
updates its body, and pins it. Idempotent: re-running just updates the same issue.

Mutating actions require --publish. Without it, this is a dry run that only
reports what it would do.

    uv run publish.py            # dry run
    uv run publish.py --publish  # create/update + pin
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from lib import config, gh  # noqa: E402


def _find_issue() -> int | None:
    # Search the repo for an open issue whose body carries our marker.
    q = f'repo:{config.REPO} is:issue is:open in:body "{config.DASHBOARD_MARKER}"'
    res = gh.api_json(f"search/issues?q={gh_quote(q)}")
    for item in res.get("items", []):
        return item["number"]

    # GitHub issue search does not index HTML comments, so fall back to scanning
    # open issues (newest first) for the marker in the body.
    out = gh._run([
        "issue", "list",
        "--repo", config.REPO,
        "--state", "open",
        "--limit", "100",
        "--json", "number,body",
    ])
    for item in json.loads(out):
        if config.DASHBOARD_MARKER in item.get("body", ""):
            return item["number"]
    return None


def gh_quote(q: str) -> str:
    from urllib.parse import quote
    return quote(q, safe="")


def _node_id(number: int) -> str:
    out = gh.api_json(f"repos/{config.REPO}/issues/{number}")
    return out["node_id"]


def _pin(number: int) -> None:
    node = _node_id(number)
    mutation = "mutation($id:ID!){pinIssue(input:{issueId:$id}){issue{number}}}"
    try:
        gh.graphql(mutation, id=node)
    except gh.GhError as exc:
        # Pinning fails if already pinned or the 3-pin limit is hit; non-fatal.
        print(f"publish: pin skipped ({exc})")


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("--publish", action="store_true", help="actually create/update + pin the issue")
    args = p.parse_args()

    if not config.ISSUE_FILE.exists():
        print("publish: out/issue.md missing; run render.py first", file=sys.stderr)
        return 1
    body = config.ISSUE_FILE.read_text()

    number = _find_issue()
    if not args.publish:
        action = f"update issue #{number}" if number else "create a new issue"
        print(f"publish: DRY RUN. Would {action} and pin it. ({len(body)} bytes)")
        print("publish: re-run with --publish to apply.")
        return 0

    if number:
        gh._run([
            "issue", "edit", str(number), "--repo", config.REPO, "--body-file", str(config.ISSUE_FILE)
        ])
        print(f"publish: updated issue #{number}")
    else:
        out = gh._run([
            "issue", "create", "--repo", config.REPO,
            "--title", config.DASHBOARD_TITLE,
            "--label", config.DASHBOARD_LABEL,
            "--body-file", str(config.ISSUE_FILE),
        ])
        number = int(out.strip().rstrip("/").split("/")[-1])
        print(f"publish: created issue #{number}")

    _pin(number)
    print(f"publish: done -> https://github.com/{config.REPO}/issues/{number}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
