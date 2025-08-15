from enum import Enum
import re
import os
from dataclasses import dataclass, asdict
import glob
import json


@dataclass
class ParsingContext:
    example_path: str
    extension: str
    comment_prefix: str


class SDKParsingContext(Enum):
    PYTHON = ParsingContext(
        example_path="sdks/python/examples", extension=".py", comment_prefix="#"
    )
    TYPESCRIPT = ParsingContext(
        example_path="sdks/typescript/src/examples/v1",
        extension=".ts",
        comment_prefix="//",
    )
    GO = ParsingContext(
        example_path="pkg/examples", extension=".go", comment_prefix="//"
    )


@dataclass
class Snippet:
    title: str
    content: str
    github_url: str
    language: str


@dataclass
class ProcessedExample:
    context: SDKParsingContext
    filepath: str
    snippets: list[Snippet]
    raw_content: str
    output_path: str


ROOT = "../../"
BASE_SNIPPETS_DIR = os.path.join(ROOT, "frontend", "docs", "lib")
OUTPUT_DIR = os.path.join(BASE_SNIPPETS_DIR, "generated", "snippets")
IGNORED_FILE_PATTERNS = [r"__init__\.py$", r"test_.*\.py$"]


def to_snake_case(text):
    text = re.sub(r"[^a-zA-Z\s\-_]", "", text)
    text = re.sub(r"[-\s]+", "_", text)
    text = re.sub(r"([a-z0-9])([A-Z])", r"\1_\2", text)
    text = re.sub(r"([A-Z])([A-Z][a-z])", r"\1_\2", text)
    text = re.sub(r"_+", "_", text)
    return text.strip("_").lower()


Title = str
Content = str


def parse_snippet_from_block(match: re.Match[str]) -> tuple[Title, Content]:
    title = to_snake_case(match.group(1).strip())
    code = match.group(2).strip()

    return title, code


def parse_snippets(ctx: SDKParsingContext, filename: str) -> list[Snippet]:
    pattern = r"# >\s+(.+?)\n(.*?)\n# !!"
    subdir = ctx.value.example_path.rstrip("/").lstrip("/")
    base_path = ROOT + subdir

    with open(filename) as f:
        content = f.read()

    github_url = f"https://github.com/hatchet-dev/hatchet/tree/main/examples/{ctx.name.lower()}{filename.replace(base_path, '')}"

    return [
        Snippet(
            title=x[0],
            content=x[1],
            github_url=github_url,
            language=ctx.name.lower(),
        )
        for match in re.finditer(pattern, content, re.DOTALL)
        if (x := parse_snippet_from_block(match))
    ]


def process_example(ctx: SDKParsingContext, filename: str) -> ProcessedExample:
    with open(filename) as f:
        content = f.read()
        return ProcessedExample(
            context=ctx,
            filepath=filename,
            output_path=f"examples/{ctx.name.lower()}{filename.replace(ROOT + ctx.value.example_path, '')}",
            snippets=parse_snippets(ctx, filename),
            raw_content=content,
        )


def process_examples(ctx: SDKParsingContext) -> list[ProcessedExample]:
    subdir = ctx.value.example_path.rstrip("/").lstrip("/")
    base_path = ROOT + subdir
    path = base_path + "/**/*" + ctx.value.extension

    return [
        process_example(ctx, filename)
        for filename in glob.iglob(path, recursive=True)
        if not any(re.search(pattern, filename) for pattern in IGNORED_FILE_PATTERNS)
    ]


def write_snippets_to_files(examples: list[ProcessedExample]) -> None:
    with open(os.path.join(BASE_SNIPPETS_DIR, "snips.ts"), "w") as f:
        f.write(
            "export type Snippet = {\n"
            "  title: string;\n"
            "  content: string;\n"
            "  github_url: string;\n"
            f"  language: {' | '.join([f"'{e.name.lower()}'" for e in SDKParsingContext])};\n"
            "}\n"
        )

    for example in examples:
        keys = (
            example.output_path.replace("examples/", "")
            .replace(example.context.value.extension, "")
            .split("/")
        )

        for snippet in example.snippets:
            file_path_parts = keys + [f"{snippet.title}.ts"]
            file_path = os.path.join(OUTPUT_DIR, *file_path_parts)

            os.makedirs(os.path.dirname(file_path), exist_ok=True)
            ts = (
                'import { Snippet } from "@/lib/snips";\n\n'
                + "export const snippet: Snippet = "
                + json.dumps(asdict(snippet), indent=2)
            )

            with open(file_path, "w") as f:
                f.write(ts)

            print(f"Wrote: {file_path}")


if __name__ == "__main__":
    processed_examples = process_examples(SDKParsingContext.PYTHON)

    write_snippets_to_files(processed_examples)
