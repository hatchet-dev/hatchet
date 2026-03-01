import glob
import json
import os
import re
from dataclasses import asdict, dataclass
from enum import Enum
from typing import Any, Callable, cast

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
    r"test_.*_spec\.rb$",
    r"spec_helper\.rb$",
    r"Gemfile",
    r"\.rspec$",
    r"README\.md$",
]

GUIDES_BASE = "sdks/guides"


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
        example_path="sdks/go/examples", extension=".go", comment_prefix="//"
    )
    RUBY = ParsingContext(
        example_path="sdks/ruby/examples", extension=".rb", comment_prefix="#"
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


@dataclass
class DocumentationPage:
    title: str
    href: str


Title = str
Content = str


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


GUIDES_LANG_TO_CTX: dict[str, SDKParsingContext] = {
    "python": SDKParsingContext.PYTHON,
    "typescript": SDKParsingContext.TYPESCRIPT,
    "go": SDKParsingContext.GO,
    "ruby": SDKParsingContext.RUBY,
}


def process_guides() -> list[ProcessedExample]:
    """Process guide examples from sdks/guides/{lang}/ into examples/{lang}/guides/."""
    examples: list[ProcessedExample] = []

    for lang_dir, ctx in GUIDES_LANG_TO_CTX.items():
        guides_base = os.path.join(ROOT, GUIDES_BASE, lang_dir)
        if not os.path.isdir(guides_base):
            continue

        pattern = guides_base + "/**/*" + ctx.value.extension

        for filename in glob.iglob(pattern, recursive=True):
            if any(re.search(p, filename) for p in IGNORED_FILE_PATTERNS):
                continue

            with open(filename) as f:
                content = f.read()

            rel_path = filename.replace(guides_base, "")
            output_path = f"examples/{ctx.name.lower()}/guides{rel_path}"
            code_path = output_path

            github_url = f"https://github.com/{OUTPUT_GITHUB_ORG}/{OUTPUT_GITHUB_REPO}/tree/main/{code_path}"

            comment_prefix = re.escape(ctx.value.comment_prefix)
            snippet_pattern = rf"{comment_prefix} >\s+(.+?)\n(.*?){comment_prefix} !!"
            matches = list(re.finditer(snippet_pattern, content, re.DOTALL))

            if not matches:
                snippets = [
                    Snippet(
                        title="all",
                        content=content,
                        githubUrl=github_url,
                        language=ctx.name.lower(),
                        codePath=code_path,
                    )
                ]
            else:
                snippets = [
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

            examples.append(
                ProcessedExample(
                    context=ctx,
                    filepath=filename,
                    output_path=output_path,
                    snippets=snippets,
                    raw_content=content,
                )
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


GUIDES_SOURCE = "sdks/guides"
GUIDES_OUTPUT = "examples"


def _read_sdk_version(lang: str) -> str:
    """Read the published SDK version from the source package file."""
    if lang == "python":
        path = os.path.join(ROOT, "sdks", "python", "pyproject.toml")
        for line in open(path):
            if line.startswith("version = "):
                return line.split('"')[1].strip()
    elif lang == "typescript":
        data = json.load(open(os.path.join(ROOT, "sdks", "typescript", "package.json")))
        return data["version"]
    elif lang == "ruby":
        path = os.path.join(ROOT, "sdks", "ruby", "src", "lib", "hatchet", "version.rb")
        for line in open(path):
            if "VERSION" in line:
                return line.split('"')[1].strip()
    elif lang == "go":
        # Go module uses monorepo; use Python SDK version as proxy for hatchet release
        return _read_sdk_version("python")
    return "0.0.0"


def copy_guide_dep_file(
    lang: str,
    filename: str,
    use_published: bool = True,
) -> None:
    """Copy a dep file from sdks/guides/{lang}/ to examples/{lang}/guides/.
    If use_published, replace local path refs with published package versions."""
    src = os.path.join(ROOT, GUIDES_SOURCE, lang, filename)
    out_dir = os.path.join(ROOT, GUIDES_OUTPUT, lang, "guides")
    if not os.path.isfile(src) or not os.path.isdir(out_dir):
        return
    content = open(src).read()

    if use_published:
        ver = _read_sdk_version(lang)
        if lang == "go":
            content = content.replace("module github.com/hatchet-dev/hatchet/sdks/guides/go", "module github.com/hatchet-dev/hatchet/examples/go/guides")
            go_ver = f"v{ver}" if not ver.startswith("v") else ver
            content = content.replace("github.com/hatchet-dev/hatchet v0.0.0", f"github.com/hatchet-dev/hatchet {go_ver}")
            content = re.sub(r"\nreplace github\.com/hatchet-dev/hatchet => \.\./\.\./\.\.\s*\n?", "\n", content)
        elif lang == "python":
            content = content.replace('hatchet-sdk = { path = "../../python", develop = true }', f'hatchet-sdk = "^{ver}"')
        elif lang == "ruby":
            content = content.replace(
                'gem "hatchet-sdk", path: "../../ruby/src"',
                f'gem "hatchet-sdk", "~> {ver}"',
            )
        elif lang == "typescript":
            content = content.replace(
                '"@hatchet-dev/typescript-sdk": "file:../../typescript"',
                f'"@hatchet-dev/typescript-sdk": "^{ver}"',
            )

    with open(os.path.join(out_dir, filename), "w") as f:
        f.write(content)


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

    # Copy dep files from sdks/guides/ to examples/*/guides/ with published SDK refs
    copy_guide_dep_file("go", "go.mod")
    copy_guide_dep_file("python", "pyproject.toml")
    copy_guide_dep_file("ruby", "Gemfile")
    copy_guide_dep_file("typescript", "package.json")


class JavaScriptObjectDecoder(json.JSONDecoder):
    def replacement(self, match: re.Match[str]) -> str:
        indent = match.group(1)
        key = match.group(2)
        return f'{indent}"{key}":'

    def decode(self, s: str, _w: Callable[..., Any] = re.compile(r"\s").match) -> Any:  # type: ignore[override]
        pattern = r"^(\s*)([a-zA-Z_$][a-zA-Z0-9_$-]*)\s*:"
        quoted = re.sub(pattern, self.replacement, s)
        result = re.sub(pattern, self.replacement, quoted, flags=re.MULTILINE)
        result = re.sub(
            r"(\{\s*)([a-zA-Z_$][a-zA-Z0-9_$-]*)\s*:",
            r'\1"\2":',
            result,
        )
        result = re.sub(r",(\s*\n?\s*})(\s*);?", r"\1", result)

        return super().decode(result)


def is_doc_page(key: str, children: str | dict[str, Any]) -> bool:
    if key.strip().startswith("--"):
        return False

    if isinstance(children, str):
        return True

    return "title" in children


def extract_doc_name(value: str | dict[str, Any]) -> str:
    if isinstance(value, str):
        return value

    if "title" in value:
        return value["title"]

    raise ValueError(f"Invalid doc value: {value}")


def keys_to_path(keys: list[str]) -> str:
    keys = [k for k in keys if k]

    if len(keys) == 0:
        return ""

    if len(keys) == 1:
        return "/" + keys[0]

    return "/" + "/".join(keys).replace("//", "/").rstrip("/")


def write_doc_index_to_app() -> None:
    docs_root = os.path.join(ROOT, "frontend", "docs")
    pages_dir = os.path.join(docs_root, "pages/")

    path = docs_root + "/**/_meta.js"
    tree: dict[str, Any] = {}

    for filename in glob.iglob(path, recursive=True):
        with open(filename) as f:
            content = f.read().replace("export default ", "").strip().rstrip(";")
            parsed_meta = cast(
                dict[str, Any], json.loads(content, cls=JavaScriptObjectDecoder)
            )

            keys = (
                filename.replace(pages_dir, "")
                .replace("_meta.js", "")
                .rstrip("/")
                .split("/")
            )
            docs = {
                key: extract_doc_name(value)
                for key, value in parsed_meta.items()
                if is_doc_page(key, value)
            }

            for key, title in docs.items():
                key = key.strip() or "index"
                full_keys = keys + [key]
                full_keys = [k for k in full_keys]

                current = tree
                for k in full_keys[:-1]:
                    k = k or "index"
                    if k not in current:
                        current[k] = {}
                    elif isinstance(current[k], str):
                        break

                    current = current[k]
                else:
                    current[full_keys[-1]] = asdict(
                        DocumentationPage(
                            title=title,
                            href=f"https://docs.hatchet.run{keys_to_path(full_keys[:-1])}/{key}",
                        )
                    )

    out_dir = os.path.join(ROOT, "frontend", "app", "src", "lib", "generated", "docs")
    os.makedirs(out_dir, exist_ok=True)

    with open(os.path.join(out_dir, "index.ts"), "w") as f:
        f.write("export const docsPages = ")
        json.dump(tree, f, indent=2)
        f.write(" as const;\n")


if __name__ == "__main__":
    processed_examples = process_examples() + process_guides()

    tree = create_snippet_tree(processed_examples)

    print(f"Writing snippets to {OUTPUT_DIR}/index.ts")
    os.makedirs(OUTPUT_DIR, exist_ok=True)

    with open(os.path.join(OUTPUT_DIR, "index.ts"), "w") as f:
        f.write("export const snippets = ")
        json.dump(tree, f, indent=2)
        f.write(" as const;\n")

    language_union = ' | '.join([f"'{v.name.lower()}'" for v in SDKParsingContext])
    snippet_type = (
        "export type Snippet = {\n"
        "    title: string;\n"
        "    content: string;\n"
        "    githubUrl: string;\n"
        "    codePath: string;\n"
        f"    language: {language_union}\n"
        "};\n"
    )

    print(f"Writing snippet type to {BASE_SNIPPETS_DIR}/snippet.ts")
    with open(os.path.join(BASE_SNIPPETS_DIR, "snippet.ts"), "w") as f:
        f.write(snippet_type)

    write_examples(processed_examples)
    write_doc_index_to_app()
