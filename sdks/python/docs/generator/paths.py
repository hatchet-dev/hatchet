import os

from docs.generator.doc_types import HAND_AUTHORED_BASENAMES, Document


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


def assert_ownership(documents: list[Document]) -> None:
    offenders = [
        d.readable_source_path
        for d in documents
        if not d.directory and d.basename in HAND_AUTHORED_BASENAMES
    ]

    if offenders:
        raise RuntimeError(
            f"Refusing to overwrite hand-authored pages: {offenders}. "
            "Rename the mkdocs source files or update HAND_AUTHORED_BASENAMES."
        )
