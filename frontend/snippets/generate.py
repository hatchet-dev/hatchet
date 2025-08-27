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
    codePath: str


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
OUTPUT_GITHUB_ORG = "hatchet-dev"
OUTPUT_GITHUB_REPO = "hatchet"
IGNORED_FILE_PATTERNS = [
    r"__init__\.py$",
    r"test_.*\.py$",
    r"\.test\.ts$",
    r"\.test-d\.ts$",
    r"test_.*\.go$",
    r"_test\.go$",
    r"\.e2e\.ts$",
]


def to_snake_case(text):
    text = re.sub(r"[^a-zA-Z0-9\s\-_]", "", text)
    text = re.sub(r"[-\s]+", "_", text)
    text = re.sub(r"([a-z0-9])([A-Z])", r"\1_\2", text)
    text = re.sub(r"([A-Z])([A-Z][a-z])", r"\1_\2", text)
    text = re.sub(r"_+", "_", text)
    return text.strip("_").lower()


Title = str
Content = str


def dedent_code(code: str) -> str:
    lines = code.split("\n")
    if not lines:
        return code

    min_indent = min((len(line) - len(line.lstrip())) for line in lines if line.strip())

    dedented_lines = [
        line[min_indent:] if len(line) >= min_indent else line for line in lines
    ]

    return "\n".join(dedented_lines).strip() + "\n"


def parse_snippet_from_block(match: re.Match[str]) -> tuple[Title, Content]:
    title = to_snake_case(match.group(1).strip())
    code = match.group(2)

    return title, dedent_code(code)


def parse_snippets(ctx: SDKParsingContext, filename: str) -> list[Snippet]:
    comment_prefix = re.escape(ctx.value.comment_prefix)
    pattern = rf"{comment_prefix} >\s+(.+?)\n(.*?){comment_prefix} !!"

    subdir = ctx.value.example_path.rstrip("/").lstrip("/")
    base_path = ROOT + subdir

    with open(filename) as f:
        content = f.read()

    code_path = f"examples/{ctx.name.lower()}{filename.replace(base_path, '')}"

    github_url = f"https://github.com/{OUTPUT_GITHUB_ORG}/{OUTPUT_GITHUB_REPO}/tree/main/{code_path}"

    matches = list(re.finditer(pattern, content, re.DOTALL))

    if not matches:
        return [
            Snippet(
                title="all",
                content=content,
                githubUrl=github_url,
                language=ctx.name.lower(),
                codePath=code_path,
            )
        ]

    return [
        Snippet(
            title=x[0],
            content=x[1],
            githubUrl=github_url,
            language=ctx.name.lower(),
            codePath=code_path,
        )
        for match in matches
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


def is_excluded_line(line: str, comment_prefix: str) -> bool:
    end_pattern = f"{comment_prefix} !!"
    return line.strip() == end_pattern or "eslint-disable" in line or "HH-" in line


def process_line_content(line: str) -> str:
    return line.replace("@hatchet/", "@hatchet-dev/typescript-sdk/")


def clean_example_content(content: str, comment_prefix: str) -> str:
    lines = content.split("\n")

    return "\n".join(
        [
            process_line_content(line)
            for line in lines
            if not is_excluded_line(line, comment_prefix)
        ]
    )


def write_examples(examples: list[ProcessedExample]) -> None:
    for example in examples:
        out_path = os.path.join(ROOT, example.output_path)
        out_dir = os.path.dirname(out_path)
        os.makedirs(out_dir, exist_ok=True)

        with open(out_path, "w") as f:
            f.write(
                clean_example_content(
                    example.raw_content, example.context.value.comment_prefix
                )
            )


if __name__ == "__main__":
    processed_examples = process_examples()

    tree = create_snippet_tree(processed_examples)

    print(f"Writing snippets to {OUTPUT_DIR}/index.ts")
    os.makedirs(OUTPUT_DIR, exist_ok=True)

    with open(os.path.join(OUTPUT_DIR, "index.ts"), "w") as f:
        f.write("export const snippets = ")
        json.dump(tree, f, indent=2)
        f.write(" as const;\n")

    snippet_type = (
        "export type Snippet = {\n"
        "    title: string;\n"
        "    content: string;\n"
        "    githubUrl: string;\n"
        "    codePath: string;\n"
        f"    language: {' | '.join([f"'{v.name.lower()}'" for v in SDKParsingContext])}\n"
        "};\n"
    )

    print(f"Writing snippet type to {BASE_SNIPPETS_DIR}/snippet.ts")
    with open(os.path.join(BASE_SNIPPETS_DIR, "snippet.ts"), "w") as f:
        f.write(snippet_type)

    write_examples(processed_examples)
