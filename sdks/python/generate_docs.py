#!/usr/bin/env python3
import os
import subprocess
import shutil
from pathlib import Path

def generate_python_docs():
    # Install pydoc-markdown if not already installed
    subprocess.run(["pip", "install", "pydoc-markdown"], check=True)
    
    # Get the absolute paths
    current_dir = Path(__file__).parent.absolute()
    sdk_dir = current_dir / "hatchet_sdk"
    docs_dir = current_dir.parent.parent / "frontend/docs/pages/sdks/python/api"
    
    # Remove existing docs directory if it exists
    if docs_dir.exists():
        shutil.rmtree(docs_dir)
    
    # Create docs directory
    docs_dir.mkdir(parents=True, exist_ok=True)
    
    # Create a temporary pydoc-markdown config
    config = """
loaders:
  - type: python
    packages:
      - hatchet_sdk
processors:
  - type: filter
    skip_empty_modules: true
  - type: smart
  - type: crossref
renderer:
  type: docusaurus
  docs_base_path: {}
  relative_output_path: .
  relative_sidebar_path: sidebar.json
  sidebar_top_level_label: ""
  markdown:
    descriptive_class_title: false
    render_toc: true
    render_module_header: true
    classdef_code_block: false
    signature_in_header: false
    add_method_class_prefix: false
    add_member_class_prefix: false
    sub_prefix: false
  source_linker:
    type: github
    repo: hatchet-dev/hatchet
""".format(docs_dir)
    
    config_file = current_dir / "pydoc-markdown.yml"
    config_file.write_text(config)
    
    try:
        # Generate documentation
        subprocess.run([
            "pydoc-markdown",
            str(config_file)
        ], check=True)
        
        print(f"Documentation generated at {docs_dir}")
    finally:
        # Clean up config file
        config_file.unlink()

if __name__ == "__main__":
    generate_python_docs() 