import os
import re
from pathlib import Path

from pydantic import BaseModel

from docs.generator.shared import TMP_GEN_PATH

GENERATOR_DIR = Path(__file__).resolve().parent
REPO_ROOT = GENERATOR_DIR.parents[3]
FRONTEND_DOCS_DIR = REPO_ROOT / "frontend" / "docs"
FRONTEND_PYTHON_REF_DIR = (
    FRONTEND_DOCS_DIR / "content" / "docs" / "reference" / "python"
)
DOCS_DIR = GENERATOR_DIR.parent
ROOT_META_PATH = FRONTEND_PYTHON_REF_DIR / "meta.json"

## Everything after this separator in `python/meta.json` is a hand-authored
## Python-specifics page. The generator preserves those entries verbatim and
## must never write or delete their mdx files.
SPECIFICS_SEPARATOR = "---Python Specifics---"

MD_EXTENSION = "md"
MDX_EXTENSION = "mdx"

ACRONYMS = {"cel"}


class Document(BaseModel):
    source_path: str
    readable_source_path: str
    mdx_output_path: str

    directory: str
    basename: str
    title: str

    @staticmethod
    def from_path(path: str) -> "Document":
        # example path /tmp/hatchet-python/docs/gen/feature-clients/runs.md

        relative_path = os.path.relpath(path, TMP_GEN_PATH)
        basename = os.path.splitext(os.path.basename(path))[0]

        title = re.sub(
            "[^0-9a-zA-Z ]+", "", basename.replace("_", " ").replace("-", " ")
        ).title()
        title = " ".join(
            word.upper() if word.lower() in ACRONYMS else word
            for word in title.split(" ")
        )

        mdx_relative_path = os.path.splitext(relative_path)[0] + "." + MDX_EXTENSION

        return Document(
            source_path=path,
            readable_source_path=relative_path,
            mdx_output_path=str(FRONTEND_PYTHON_REF_DIR / mdx_relative_path),
            directory=os.path.dirname(relative_path),
            basename=basename,
            title=title,
        )
