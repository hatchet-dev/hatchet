# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""Classification interface (the one non-deterministic stage).

The agent uses this to find failure signatures that have never been classified
and to record a cause for each. Everything here is deterministic plumbing; the
only judgement call is choosing the category + reason, which the agent supplies.
Results are cached by signature in .cache/classifications.json, so steady-state
runs do little or no LLM work.

Usage:
    uv run classify.py pending                       # JSON list of unclassified signatures
    uv run classify.py set --hash H --category C --reason "..."   # upsert one
    uv run classify.py stats                         # coverage summary
"""

from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from lib import cache, config  # noqa: E402


def _pending() -> list[dict]:
    analysis = cache.read_json(config.ANALYSIS_FILE, default={}) or {}
    sigs = analysis.get("signatures", {})
    done = cache.load_classifications()
    pending = []
    for h, info in sigs.items():
        if h not in done:
            pending.append({"sig_hash": h, **info})
    return pending


def cmd_pending(_args) -> int:
    print(json.dumps(_pending(), indent=2))
    return 0


def cmd_set(args) -> int:
    if args.category not in config.CATEGORIES:
        print(f"error: category must be one of {config.CATEGORIES}", file=sys.stderr)
        return 2
    data = cache.load_classifications()
    analysis = cache.read_json(config.ANALYSIS_FILE, default={}) or {}
    sig = analysis.get("signatures", {}).get(args.hash, {})
    data[args.hash] = {
        "signature": sig.get("signature"),
        "category": args.category,
        "reason": args.reason,
        "model": args.model,
        "classified_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    }
    cache.save_classifications(data)
    print(f"classified {args.hash}: {args.category}")
    return 0


def cmd_stats(_args) -> int:
    analysis = cache.read_json(config.ANALYSIS_FILE, default={}) or {}
    total = len(analysis.get("signatures", {}))
    pending = len(_pending())
    print(json.dumps({"total_signatures": total, "pending": pending, "classified": total - pending}))
    return 0


def main() -> int:
    p = argparse.ArgumentParser()
    sub = p.add_subparsers(dest="cmd", required=True)
    sub.add_parser("pending").set_defaults(func=cmd_pending)
    sub.add_parser("stats").set_defaults(func=cmd_stats)
    s = sub.add_parser("set")
    s.add_argument("--hash", required=True)
    s.add_argument("--category", required=True)
    s.add_argument("--reason", required=True)
    s.add_argument("--model", default="cursor-agent")
    s.set_defaults(func=cmd_set)
    args = p.parse_args()
    return args.func(args)


if __name__ == "__main__":
    raise SystemExit(main())
