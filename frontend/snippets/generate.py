import glob
import json
import os
import re
from dataclasses import asdict, dataclass
from enum import Enum
from typing import Any


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
        example_path="sdks/typescript/src/v1/examples",
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
    githubUrl: str
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
    text = re.sub(r"[^a-zA-Z0-9\s\-_]", "", text)
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
    comment_prefix = re.escape(ctx.value.comment_prefix)  # Escape for regex safety
    pattern = rf"{comment_prefix} >\s+(.+?)\n(.*?)\n{comment_prefix} !!"

    subdir = ctx.value.example_path.rstrip("/").lstrip("/")
    base_path = ROOT + subdir

    with open(filename) as f:
        content = f.read()

    github_url = f"https://github.com/hatchet-dev/hatchet/tree/main/examples/{ctx.name.lower()}{filename.replace(base_path, '')}"

    return [
        Snippet(
            title=x[0],
            content=x[1],
            githubUrl=github_url,
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


def process_examples() -> list[ProcessedExample]:
    examples: list[ProcessedExample] = []

    for ctx in SDKParsingContext:
        subdir = ctx.value.example_path.rstrip("/").lstrip("/")
        base_path = ROOT + subdir
        path = base_path + "/**/*" + ctx.value.extension

        examples.extend(
            [
                process_example(ctx, filename)
                for filename in glob.iglob(path, recursive=True)
                if not any(
                    re.search(pattern, filename) for pattern in IGNORED_FILE_PATTERNS
                )
            ]
        )

    return examples


def create_snippet_tree(examples: list[ProcessedExample]) -> dict[str, dict[str, Any]]:
    tree: dict[str, Any] = {}

    for example in examples:
        keys = (
            example.output_path.replace("examples/", "")
            .replace(example.context.value.extension, "")
            .split("/")
        )

        for snippet in example.snippets:
            full_keys = keys + [snippet.title]

            current = tree
            for key in full_keys[:-1]:
                key = to_snake_case(key)
                if key not in current:
                    current[key] = {}
                current = current[key]

            current[full_keys[-1]] = asdict(snippet)

    return tree


if __name__ == "__main__":
    processed_examples = process_examples()

    tree = create_snippet_tree(processed_examples)

    print(f"Writing snippets to {OUTPUT_DIR}/index.ts")

    with open(os.path.join(OUTPUT_DIR, "index.ts"), "w") as f:
        f.write("export const snippets = ")
        json.dump(tree, f, indent=2)
        f.write(" as const;\n")

    snippet_type = (
        "export type Snippet = {\n"
        "    title: string;\n"
        "    content: string;\n"
        "    githubUrl: string;\n"
        f"    language: {' | '.join([f"'{v.name.lower()}'" for v in SDKParsingContext])}\n"
        "};\n"
    )

    print(f"Writing snippet type to {BASE_SNIPPETS_DIR}/snippet.ts")
    with open(os.path.join(BASE_SNIPPETS_DIR, "snippet.ts"), "w") as f:
        f.write(snippet_type)
