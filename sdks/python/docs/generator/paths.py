import os

from docs.generator.doc_types import Document


def crawl_directory(directory: str, only_include: list[str]) -> list[Document]:
    return sorted(
        (
            d
            for root, _, filenames in os.walk(directory)
            for filename in filenames
            if (
                d := Document.from_path(os.path.join(root, filename))
            ).readable_source_path
            in only_include
            or not only_include
        ),
        key=lambda d: d.readable_source_path,
    )


def assert_ownership(documents: list[Document], specifics: set[str]) -> None:
    offenders = [
        d.readable_source_path
        for d in documents
        if not d.directory and d.basename in specifics
    ]

    if offenders:
        raise RuntimeError(
            f"Refusing to overwrite hand-authored Python-specifics pages: {offenders}. "
            "Rename the mkdocs source files or the specifics pages."
        )
