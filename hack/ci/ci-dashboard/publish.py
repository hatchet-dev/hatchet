# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""Stage 6: publish out/issue.md to the canonical dashboard issue.

Updates a specific issue in place (config.DASHBOARD_ISSUE, default #4204) and
pins it. Idempotent: re-running just overwrites the same issue body.

Mutating actions require --publish. Without it, this is a dry run that only
reports what it would do.

    uv run publish.py                 # dry run against the canonical issue
    uv run publish.py --publish       # update + pin #4204
    uv run publish.py --publish --issue 1234            # target a different issue
    uv run publish.py --publish --body-file staging/issue.md   # publish a staged body
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from lib import config, gh  # noqa: E402


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
    p.add_argument("--publish", action="store_true", help="actually update + pin the issue")
    p.add_argument("--issue", type=int, default=config.DASHBOARD_ISSUE,
                   help=f"issue number to update (default {config.DASHBOARD_ISSUE})")
    p.add_argument("--body-file", type=Path, default=config.ISSUE_FILE,
                   help=f"rendered issue body to publish (default {config.ISSUE_FILE})")
    args = p.parse_args()

    body_file: Path = args.body_file
    if not body_file.exists():
        print(f"publish: {body_file} missing; run render.py (or run.sh --stage) first", file=sys.stderr)
        return 1
    body = body_file.read_text()
    number = args.issue

    if not args.publish:
        print(f"publish: DRY RUN. Would update issue #{number} from {body_file} and pin it. "
              f"({len(body)} bytes)")
        print(f"publish: target -> https://github.com/{config.REPO}/issues/{number}")
        print("publish: re-run with --publish to apply.")
        return 0

    gh._run([
        "issue", "edit", str(number), "--repo", config.REPO, "--body-file", str(body_file)
    ])
    print(f"publish: updated issue #{number}")
    _pin(number)
    print(f"publish: done -> https://github.com/{config.REPO}/issues/{number}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
