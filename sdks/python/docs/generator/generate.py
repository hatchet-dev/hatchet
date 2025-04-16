import asyncio
import os
import re
import shutil
from dataclasses import dataclass
from typing import Coroutine, ParamSpec, TypeVar

from openai import AsyncOpenAI
from openai.types.chat import (
    ChatCompletionMessageParam,
    ChatCompletionSystemMessageParam,
    ChatCompletionUserMessageParam,
)
from docs.generator.shared import TMP_GEN_PATH

T = TypeVar("T")
P = ParamSpec("P")
R = TypeVar("R")

key = os.environ["OPENAI_API_KEY"]

client = AsyncOpenAI(api_key=key)


async def gather_max_concurrency(
    *tasks: Coroutine[None, None, T],
    max_concurrency: int,
) -> list[T]:
    """asyncio.gather with cap on subtasks executing at once."""
    sem = asyncio.Semaphore(max_concurrency)

    async def task_wrapper(task: Coroutine[None, None, T]) -> T:
        async with sem:
            return await task

    return await asyncio.gather(
        *(task_wrapper(task) for task in tasks),
        return_exceptions=False,
    )


def crawl_directory(directory: str) -> list[str]:
    return [
        os.path.join(root, filename)
        for root, _, filenames in os.walk(directory)
        for filename in filenames
    ]


async def clean_markdown_with_openai(file_path: str) -> None:
    print("Generating mdx for", file_path)

    with open(file_path, "r", encoding="utf-8") as f:
        original_md = f.read()

    system_prompt = (
        "Clean up this markdown file to make it correct and more readable and using via mdx. "
        "Strip any unnecessary paragraph characters, but do not change any actual code, sentences, or content:"
        "Return only the content. You should not enclode the markdown in backticks or any other formatting. "
        "You must ensure that the MDX you generate will build correctly using our build tool: Nextra, which means "
        "it must not have any invalid characters or anything else that would cause the build to crash. "
        "One such example would be to avoid this error `Could not parse expression with acorn`. Make sure you follow the MDX troubleshooting guide."
    )
    messages: list[ChatCompletionMessageParam] = [
        ChatCompletionSystemMessageParam(content=system_prompt, role="system"),
        ChatCompletionUserMessageParam(content=original_md, role="user"),
    ]

    response = await client.chat.completions.create(
        model="gpt-4o", messages=messages, temperature=0
    )

    content = response.choices[0].message.content

    if not content:
        return None

    out_path = file_path.replace(
        TMP_GEN_PATH, "../../frontend/docs/pages/sdks/python"
    ).replace(".md", ".mdx")

    with open(out_path, "w", encoding="utf-8") as f:
        f.write(content)


def rm_rf(path: str) -> None:
    shutil.rmtree(path, ignore_errors=True)


@dataclass
class DocMetadata:
    prefix: str
    key: str
    title: str


def generate_single_meta_entry(doc: DocMetadata) -> str:
    file_key = doc.key.replace(".md", "")

    return f"""
        "{file_key}": {{
            "title": "{doc.title}",
        }},
    """


def generate_sub_meta_entry(child: str) -> str:
    child = child.replace("/", "")
    return f"""
        "{child}": {{
            "title": "{child.title()}",
        }},
    """


def generate_meta_js(docs: list[DocMetadata], children: set[str]) -> str:
    prefix = docs[0].prefix

    entries = "".join(
        [generate_single_meta_entry(doc) for doc in docs]
        + [generate_sub_meta_entry(child.replace(prefix, "")) for child in children]
    )

    return f"export default {{{entries}}}"


def generate_doc_metadata(path: str) -> DocMetadata:
    prefix, key = path.rsplit("/", maxsplit=1)

    doc_title = re.sub(
        "[^0-9a-zA-Z ]+", "", key.replace(".md", "").replace("_", " ").replace("-", " ")
    ).title()

    return DocMetadata(prefix=prefix, key=key, title=doc_title)


def find_child_paths(prefix: str, docs: list[DocMetadata]) -> set[str]:
    return {
        doc.prefix
        for doc in docs
        if doc.prefix.startswith(prefix)
        and doc.prefix != prefix
        and doc.prefix.count("/") == prefix.count("/") + 1
    }


async def run() -> None:
    rm_rf(TMP_GEN_PATH)

    try:
        os.system("poetry run mkdocs build")
        files = crawl_directory(TMP_GEN_PATH)

        await gather_max_concurrency(
            *[clean_markdown_with_openai(f) for f in files], max_concurrency=10
        )

        doc_metadata = [generate_doc_metadata(file) for file in files]

        prefixes = {f.prefix for f in doc_metadata}

        for prefix in prefixes:
            children = find_child_paths(prefix, doc_metadata)

            meta = generate_meta_js(
                [f for f in doc_metadata if f.prefix == prefix], children
            )
            out_path = (
                prefix.replace(
                    TMP_GEN_PATH, "../../frontend/docs/pages/sdks/python"
                ).replace(".md", ".mdx")
                + "/_meta.js"
            )

            with open(out_path, "w", encoding="utf-8") as f:
                f.write(meta)

        os.chdir("../../frontend/docs")
        os.system("pnpm lint:fix")
    finally:
        rm_rf("docs/site")
        rm_rf("site")
        rm_rf(TMP_GEN_PATH)

def main() -> None:
    asyncio.run(run())


if __name__ == "__main__":
    main()
