import argparse
import asyncio
import inspect
import json
import os
import subprocess
import sys
import typing
from collections import defaultdict
from pathlib import Path

from docs.generator.doc_types import (
    DOCS_DIR,
    FRONTEND_DOCS_DIR,
    FRONTEND_PYTHON_REF_DIR,
    MANIFEST_PATH,
    ROOT_META_PATH,
    SPECIFICS_SEPARATOR,
    Document,
)
from docs.generator.llm import parse_markdown, settings
from docs.generator.paths import assert_ownership, crawl_directory
from docs.generator.shared import TMP_GEN_PATH
from docs.generator.utils import gather_max_concurrency, rm_rf


def discover_feature_clients() -> dict[str, str]:
    """Map Hatchet client property names to mkdocstrings directives, e.g. `runs` -> `features.runs.RunsClient`."""
    from hatchet_sdk import Hatchet

    clients: dict[str, str] = {}

    for name, member in inspect.getmembers(Hatchet, lambda m: isinstance(m, property)):
        if not member.fget:
            continue

        returns = typing.get_type_hints(member.fget).get("return")

        if inspect.isclass(returns) and returns.__module__.startswith(
            "hatchet_sdk.features."
        ):
            module = returns.__module__.removeprefix("hatchet_sdk.")
            clients[name] = f"{module}.{returns.__name__}"

    return clients


def ensure_feature_client_stubs() -> None:
    for name, directive in sorted(discover_feature_clients().items()):
        stub_path = DOCS_DIR / "feature-clients" / f"{name}.md"

        if stub_path.exists():
            continue

        title = name.replace("_", " ").title()
        stub_path.write_text(f"# {title} Client\n\n::: {directive}\n", encoding="utf-8")
        print("Auto-created mkdocs stub for new feature client:", stub_path)


def load_root_meta() -> dict[str, typing.Any]:
    meta: dict[str, typing.Any] = json.loads(ROOT_META_PATH.read_text(encoding="utf-8"))

    if SPECIFICS_SEPARATOR not in meta.get("pages", []):
        raise RuntimeError(
            f"{ROOT_META_PATH} is missing the '{SPECIFICS_SEPARATOR}' separator. "
            "Refusing to touch it."
        )

    return meta


def specifics_pages(meta: dict[str, typing.Any]) -> list[str]:
    pages: list[str] = meta["pages"]

    return [
        p
        for p in pages[pages.index(SPECIFICS_SEPARATOR) + 1 :]
        if not p.startswith("---")
    ]


def merge_root_meta(documents: list[Document], meta: dict[str, typing.Any]) -> None:
    """Merge generator-owned entries into python/meta.json, preserving the
    hand-maintained order and everything from the specifics separator onward."""
    pages: list[str] = meta["pages"]
    separator_index = pages.index(SPECIFICS_SEPARATOR)

    emitted = sorted(
        {
            (d.directory.split(os.sep)[0] if d.directory else d.basename)
            for d in documents
            if os.path.exists(d.mdx_output_path)
        }
    )

    owned = [p for p in pages[:separator_index] if p in emitted]
    owned += [p for p in emitted if p not in owned]

    meta["pages"] = owned + pages[separator_index:]

    ROOT_META_PATH.write_text(json.dumps(meta, indent=2) + "\n", encoding="utf-8")


def assert_pages_reachable(
    documents: list[Document], meta: dict[str, typing.Any]
) -> None:
    missing_specifics = [
        p
        for p in specifics_pages(meta)
        if not (FRONTEND_PYTHON_REF_DIR / f"{p}.mdx").exists()
    ]

    if missing_specifics:
        raise RuntimeError(
            f"Hand-authored pages listed in {ROOT_META_PATH} are missing on disk: "
            f"{missing_specifics}"
        )

    root_pages = set(meta["pages"])
    unreachable = []

    for d in documents:
        if not os.path.exists(d.mdx_output_path):
            continue

        if d.directory:
            sub_meta_path = Path(os.path.dirname(d.mdx_output_path)) / "meta.json"
            sub_pages = (
                json.loads(sub_meta_path.read_text(encoding="utf-8"))["pages"]
                if sub_meta_path.exists()
                else []
            )
            reachable = (
                d.basename in sub_pages and d.directory.split(os.sep)[0] in root_pages
            )
        else:
            reachable = d.basename in root_pages

        if not reachable:
            unreachable.append(d.readable_source_path)

    if unreachable:
        raise RuntimeError(
            f"Generated mdx pages are unreachable from any meta.json: {unreachable}"
        )


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
    ## Subdirectory meta.json files (e.g. feature-clients). Only pages whose mdx
    ## exists are listed, so a freshly auto-stubbed page joins the nav once its
    ## mdx has actually been generated.
    docs_by_directory: defaultdict[str, list[Document]] = defaultdict(list)

    for document in documents:
        if document.directory and os.path.exists(document.mdx_output_path):
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
        ensure_feature_client_stubs()

        subprocess.run(["poetry", "run", "mkdocs", "build"], check=True)

        documents = crawl_directory(TMP_GEN_PATH, selections)
        root_meta = load_root_meta()
        assert_ownership(documents, set(specifics_pages(root_meta)))

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
            merge_root_meta(documents, root_meta)
            assert_pages_reachable(documents, root_meta)

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
