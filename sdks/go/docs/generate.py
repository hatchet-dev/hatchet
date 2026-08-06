#!/usr/bin/env python3
"""Generates MDX reference docs for the Go SDK into
frontend/docs/pages/reference/go, mirroring the other SDK doc generators.
Uses gomarkdoc, then post-processes the markdown into MDX-safe pages with
Nextra _meta.js navigation.

Usage: cd sdks/go && python3 docs/generate.py
"""

import re
import shutil
import subprocess
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
REPO_ROOT = SCRIPT_DIR.parents[2]
OUTPUT_DIR = REPO_ROOT / "frontend/docs/pages/reference/go"
TMP_GEN_PATH = Path("/tmp/hatchet-go/docs/gen")
GOMARKDOC_VERSION = "v1.1.0"

PACKAGES = [
    {"path": "./sdks/go", "key": "index", "title": "Overview", "h1": "Hatchet Go SDK Reference"},
    {"path": "./sdks/go/features", "key": "features", "title": "Features", "h1": "Feature Clients"},
    {"path": "./sdks/go/opentelemetry", "key": "opentelemetry", "title": "OpenTelemetry", "h1": "OpenTelemetry"},
]

SEGMENT_PATTERN = re.compile(r"(`[^`]*`|<a name=\"[^\"]*\"></a>)")


def escape_mdx(line: str) -> str:
    segments = []
    for segment in SEGMENT_PATTERN.split(line):
        if segment.startswith(("`", '<a name="')):
            segments.append(segment)
        else:
            segment = re.sub(r"\]\(<([^>]*)>\)", r"](\1)", segment)
            segments.append(re.sub(r"(?<!\\)([<{}])", r"\\\1", segment))
    return "".join(segments)


def to_mdx(content: str, title: str) -> str:
    content = re.sub(r"<!--.*?-->\n?", "", content, flags=re.DOTALL)
    content = re.sub(r"^# .*$", f"# {title}", content, count=1, flags=re.MULTILINE)
    content = re.sub(r"^## Index\n.*?(?=^#{1,2} )", "", content, flags=re.MULTILINE | re.DOTALL)
    content = re.sub(r"<a name=\"[^\"]*\"></a>\n?", "", content)

    lines = []
    in_fence = False
    for line in content.splitlines(keepends=True):
        if line.lstrip().startswith("```"):
            in_fence = not in_fence
            lines.append(line)
        elif in_fence:
            lines.append(line)
        else:
            if line.startswith("#"):
                line = re.sub(r"\[([^\]]*)\]\([^)]*\)", r"\1", line)
                if line.startswith("### func"):
                    line = line[1:]
            lines.append(escape_mdx(line))
    return "".join(lines)


def main() -> None:
    shutil.rmtree(TMP_GEN_PATH, ignore_errors=True)
    TMP_GEN_PATH.mkdir(parents=True)
    shutil.rmtree(OUTPUT_DIR, ignore_errors=True)
    OUTPUT_DIR.mkdir(parents=True)

    for pkg in PACKAGES:
        md_path = TMP_GEN_PATH / f"{pkg['key']}.md"
        subprocess.run(
            [
                "go",
                "run",
                f"github.com/princjef/gomarkdoc/cmd/gomarkdoc@{GOMARKDOC_VERSION}",
                "--output",
                str(md_path),
                pkg["path"],
            ],
            cwd=REPO_ROOT,
            check=True,
        )
        (OUTPUT_DIR / f"{pkg['key']}.mdx").write_text(
            to_mdx(md_path.read_text(encoding="utf-8"), pkg["h1"]), encoding="utf-8"
        )

    entries = "".join(
        f'  "{pkg["key"]}": {{\n    title: "{pkg["title"]}",\n    theme: {{\n      toc: true,\n    }},\n  }},\n'
        for pkg in PACKAGES
    )
    (OUTPUT_DIR / "_meta.js").write_text(f"export default {{\n{entries}}};\n", encoding="utf-8")

    shutil.rmtree(TMP_GEN_PATH)
    print(f"Wrote {len(PACKAGES)} pages to {OUTPUT_DIR}")


if __name__ == "__main__":
    main()
