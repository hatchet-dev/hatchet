import argparse
import asyncio
import json
import os
import subprocess
import sys
from collections import defaultdict

from docs.generator.doc_types import FRONTEND_DOCS_DIR, MANIFEST_PATH, Document
from docs.generator.llm import parse_markdown, settings
from docs.generator.paths import assert_ownership, crawl_directory
from docs.generator.shared import TMP_GEN_PATH
from docs.generator.utils import gather_max_concurrency, rm_rf


def load_manifest() -> dict[str, str]:
    if not MANIFEST_PATH.exists():
        return {}

    manifest: dict[str, str] = json.loads(MANIFEST_PATH.read_text(encoding="utf-8"))

    return manifest


def write_manifest(manifest: dict[str, str]) -> None:
    MANIFEST_PATH.write_text(
        json.dumps(manifest, indent=2, sort_keys=True) + "\n", encoding="utf-8"
    )


async def clean_markdown_with_openai(document: Document) -> None:
    print("Generating mdx for", document.readable_source_path)

    with open(document.source_path, "r", encoding="utf-8") as f:
        original_md = f.read()

    content = await parse_markdown(original_markdown=original_md)

    if not content:
        raise RuntimeError(f"Empty LLM response for {document.readable_source_path}")

    os.makedirs(os.path.dirname(document.mdx_output_path), exist_ok=True)

    with open(document.mdx_output_path, "w", encoding="utf-8") as f:
        f.write(f'---\ntitle: "{document.title}"\n---\n\n{content.strip()}\n')


def update_meta_json(documents: list[Document]) -> None:
    ## Only subdirectory meta.json files (e.g. feature-clients) are generated.
    ## The top-level python/meta.json is hand-maintained and must not be touched.
    docs_by_directory: defaultdict[str, list[Document]] = defaultdict(list)

    for document in documents:
        if document.directory:
            docs_by_directory[document.directory].append(document)

    for directory, docs in sorted(docs_by_directory.items()):
        meta = {
            "pages": sorted(d.basename for d in docs),
            "title": directory.replace("-", " ").title(),
        }

        meta_path = os.path.join(os.path.dirname(docs[0].mdx_output_path), "meta.json")

        with open(meta_path, "w", encoding="utf-8") as f:
            json.dump(meta, f, indent=2)
            f.write("\n")


async def run(selections: list[str], seed_manifest: bool) -> None:
    rm_rf(TMP_GEN_PATH)

    try:
        subprocess.run(["poetry", "run", "mkdocs", "build"], check=True)

        documents = crawl_directory(TMP_GEN_PATH, selections)
        assert_ownership(documents)

        manifest = load_manifest()
        hashes = {d.readable_source_path: d.source_hash() for d in documents}

        if seed_manifest:
            stale = []
        elif selections:
            stale = documents
        else:
            stale = [
                d
                for d in documents
                if manifest.get(d.readable_source_path)
                != hashes[d.readable_source_path]
                or not os.path.exists(d.mdx_output_path)
            ]

        for document in documents:
            if document not in stale:
                print("Skipping (unchanged):", document.readable_source_path)

        if stale:
            if settings.openai_api_key in ("", "fake-key"):
                sys.exit(
                    "OPENAI_API_KEY is not set (or is a placeholder), but "
                    f"{len(stale)} doc(s) need regeneration. Refusing to continue."
                )

            await gather_max_concurrency(
                *[clean_markdown_with_openai(d) for d in stale], max_concurrency=10
            )

            subprocess.run(
                ["npx", "prettier", "--write", *[d.mdx_output_path for d in stale]],
                check=True,
                cwd=str(FRONTEND_DOCS_DIR),
            )

        if not selections:
            update_meta_json(documents)

        manifest.update(hashes)
        write_manifest(manifest)
    finally:
        rm_rf("site")
        rm_rf(TMP_GEN_PATH)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--select",
        type=str,
        help="Comma-separated list of doc names to regenerate regardless of the manifest (e.g. client,feature-clients/runs).",
    )
    parser.add_argument(
        "--seed-manifest",
        action="store_true",
        help="Record hashes of the current mkdocs export in the manifest without calling OpenAI, treating the existing mdx as up to date.",
    )

    args = parser.parse_args()

    selections = (
        [f"{name.strip()}.md" for name in args.select.split(",")] if args.select else []
    )

    asyncio.run(run(selections, args.seed_manifest))


if __name__ == "__main__":
    main()
