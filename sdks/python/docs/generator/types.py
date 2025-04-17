import os
import re

from pydantic import BaseModel

from docs.generator.shared import TMP_GEN_PATH

FRONTEND_DOCS_RELATIVE_PATH = "../../frontend/docs/pages/sdks/python"

MD_EXTENSION = "md"
MDX_EXTENSION = "mdx"
PY_EXTENSION = "py"


class Document(BaseModel):
    source_path: str
    readable_source_path: str
    mdx_output_path: str
    mdx_output_meta_js_path: str

    directory: str
    basename: str

    title: str = ""
    meta_js_entry: str = ""

    @staticmethod
    def from_path(path: str) -> "Document":
        # example path /tmp/hatchet-python/docs/gen/runnables.md

        basename = os.path.splitext(os.path.basename(path))[0]
        title = re.sub(
            "[^0-9a-zA-Z ]+", "", basename.replace("_", " ").replace("-", " ")
        ).title()

        mdx_out_path = path.replace(
            TMP_GEN_PATH, "../../frontend/docs/pages/sdks/python"
        )
        mdx_out_dir = os.path.dirname(mdx_out_path)

        return Document(
            directory=os.path.dirname(path).replace(TMP_GEN_PATH, ""),
            basename=basename,
            title=title,
            meta_js_entry=f"""
                "{basename}": {{
                    "title": "{title}",
                }},
            """,
            source_path=path,
            readable_source_path=path.replace(TMP_GEN_PATH, "")[1:],
            mdx_output_path=mdx_out_path.replace(".md", ".mdx"),
            mdx_output_meta_js_path=mdx_out_dir + "/_meta.js",
        )
