import os

from docs.generator.types import Document


def crawl_directory(directory: str) -> list[Document]:
    return [
        Document.from_path(os.path.join(root, filename))
        for root, _, filenames in os.walk(directory)
        for filename in filenames
    ]


def find_child_paths(prefix: str, docs: list[Document]) -> set[str]:
    return {
        doc.directory
        for doc in docs
        if doc.directory.startswith(prefix)
        and doc.directory != prefix
        and doc.directory.count("/") == prefix.count("/") + 1
    }
