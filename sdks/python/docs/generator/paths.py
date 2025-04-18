import os

from docs.generator.types import Document


def crawl_directory(directory: str, only_include: list[str]) -> list[Document]:
    return [
        d
        for root, _, filenames in os.walk(directory)
        for filename in filenames
        if (d := Document.from_path(os.path.join(root, filename))).readable_source_path
        in only_include
        or not only_include
    ]


def find_child_paths(prefix: str, docs: list[Document]) -> set[str]:
    return {
        doc.directory
        for doc in docs
        if doc.directory.startswith(prefix)
        and doc.directory != prefix
        and doc.directory.count("/") == prefix.count("/") + 1
    }
