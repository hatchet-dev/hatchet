import hashlib
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
MANIFEST_PATH = GENERATOR_DIR / "manifest.json"

## Hand-authored pages living alongside the generated ones. The generator must
## never write or delete these, nor the hand-maintained `python/meta.json`.
HAND_AUTHORED_BASENAMES = {
    "asyncio",
    "pydantic",
    "lifespans",
    "dependency-injection",
    "dataclasses",
}

MD_EXTENSION = "md"
MDX_EXTENSION = "mdx"


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

        mdx_relative_path = os.path.splitext(relative_path)[0] + "." + MDX_EXTENSION

        return Document(
            source_path=path,
            readable_source_path=relative_path,
            mdx_output_path=str(FRONTEND_PYTHON_REF_DIR / mdx_relative_path),
            directory=os.path.dirname(relative_path),
            basename=basename,
            title=title,
        )

    def source_hash(self) -> str:
        with open(self.source_path, "rb") as f:
            return hashlib.sha256(f.read()).hexdigest()
