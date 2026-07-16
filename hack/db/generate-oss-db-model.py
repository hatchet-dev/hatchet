#!/usr/bin/env python3
"""Generate a standalone HTML visualization of the hatchet-oss database model."""

from __future__ import annotations

import argparse
import json
import re
from dataclasses import dataclass, field
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
MODEL_NAME = "hatchet-oss"
SQLC_CONFIG = ROOT / "pkg/repository/sqlcv1/sqlc.yaml"
OSS_SCHEMA_DIR = ROOT / "sql/schema"
DEFAULT_OUTPUT = ROOT / f"generated/{MODEL_NAME}.html"
HTML_TEMPLATE = ROOT / "hack/db/hatchet-oss-template.html"
IDENTIFIER = r'(?:"(?:[^"]|"")*"|[A-Za-z_][A-Za-z0-9_$]*)(?:\.(?:"(?:[^"]|"")*"|[A-Za-z_][A-Za-z0-9_$]*))?'


@dataclass
class Column:
    name: str
    data_type: str
    nullable: bool
    default: str | None = None
    primary_key: bool = False
    foreign_key: bool = False


@dataclass(frozen=True)
class ForeignKey:
    columns: tuple[str, ...]
    target_table: str
    target_columns: tuple[str, ...]
    name: str | None = None


@dataclass
class Table:
    name: str
    source: str
    columns: list[Column] = field(default_factory=list)
    primary_key: tuple[str, ...] = ()
    foreign_keys: list[ForeignKey] = field(default_factory=list)


def discover_oss_schema_paths(config_path: Path = SQLC_CONFIG) -> tuple[Path, ...]:
    lines = config_path.read_text(encoding="utf-8").splitlines()
    schema_paths: list[Path] = []
    schema_indent: int | None = None

    for line in lines:
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            continue
        indent = len(line) - len(line.lstrip())
        if schema_indent is None:
            if stripped == "schema:":
                schema_indent = indent
            continue
        if indent <= schema_indent:
            break

        match = re.match(r"-\s+(.+?)\s*$", stripped)
        if not match:
            continue
        configured_path = match.group(1).strip("\"'")
        path = (config_path.parent / configured_path).resolve()
        try:
            path.relative_to(OSS_SCHEMA_DIR)
        except ValueError:
            continue
        if path.name != "pg-stubs.sql":
            schema_paths.append(path)

    if not schema_paths:
        raise ValueError(f"no OSS schema snapshots found in {config_path}")
    return tuple(schema_paths)


def unquote_identifier(value: str) -> str:
    return ".".join(
        part[1:-1].replace('""', '"') if part.startswith('"') else part
        for part in value.split(".")
    )


def display_path(path: Path) -> str:
    try:
        return str(path.relative_to(ROOT))
    except ValueError:
        return str(path)


def strip_comments(sql: str) -> str:
    result: list[str] = []
    index = 0
    quote: str | None = None
    while index < len(sql):
        char = sql[index]
        next_char = sql[index + 1] if index + 1 < len(sql) else ""
        if quote:
            result.append(char)
            if char == quote:
                if next_char == quote:
                    result.append(next_char)
                    index += 1
                else:
                    quote = None
        elif char in ("'", '"'):
            quote = char
            result.append(char)
        elif char == "-" and next_char == "-":
            while index < len(sql) and sql[index] != "\n":
                index += 1
            result.append("\n")
        elif char == "/" and next_char == "*":
            index += 2
            while index + 1 < len(sql) and sql[index : index + 2] != "*/":
                if sql[index] == "\n":
                    result.append("\n")
                index += 1
            index += 1
        else:
            result.append(char)
        index += 1
    return "".join(result)


def find_closing_parenthesis(sql: str, start: int) -> int:
    depth = 0
    index = start
    quote: str | None = None
    dollar_tag: str | None = None
    while index < len(sql):
        if dollar_tag:
            if sql.startswith(dollar_tag, index):
                index += len(dollar_tag)
                dollar_tag = None
                continue
            index += 1
            continue

        char = sql[index]
        if quote:
            if char == quote:
                if index + 1 < len(sql) and sql[index + 1] == quote:
                    index += 2
                    continue
                quote = None
        elif char in ("'", '"'):
            quote = char
        elif char == "$":
            match = re.match(r"\$[A-Za-z0-9_]*\$", sql[index:])
            if match:
                dollar_tag = match.group(0)
                index += len(dollar_tag)
                continue
        elif char == "(":
            depth += 1
        elif char == ")":
            depth -= 1
            if depth == 0:
                return index
        index += 1
    raise ValueError("unclosed parenthesis in SQL schema")


def split_top_level(value: str) -> list[str]:
    parts: list[str] = []
    start = 0
    depth = 0
    quote: str | None = None
    index = 0
    while index < len(value):
        char = value[index]
        if quote:
            if char == quote:
                if index + 1 < len(value) and value[index + 1] == quote:
                    index += 1
                else:
                    quote = None
        elif char in ("'", '"'):
            quote = char
        elif char in "([":
            depth += 1
        elif char in ")]":
            depth -= 1
        elif char == "," and depth == 0:
            parts.append(value[start:index].strip())
            start = index + 1
        index += 1
    tail = value[start:].strip()
    if tail:
        parts.append(tail)
    return parts


def parse_identifiers(value: str) -> tuple[str, ...]:
    return tuple(
        unquote_identifier(item.strip())
        for item in split_top_level(value)
        if item.strip()
    )


def parse_foreign_key(fragment: str) -> ForeignKey | None:
    match = re.search(
        rf"(?:CONSTRAINT\s+({IDENTIFIER})\s+)?FOREIGN\s+KEY\s*"
        rf"\((.*?)\)\s+REFERENCES\s+({IDENTIFIER})\s*\((.*?)\)",
        fragment,
        re.IGNORECASE | re.DOTALL,
    )
    if not match:
        return None
    return ForeignKey(
        columns=parse_identifiers(match.group(2)),
        target_table=unquote_identifier(match.group(3)),
        target_columns=parse_identifiers(match.group(4)),
        name=unquote_identifier(match.group(1)) if match.group(1) else None,
    )


def parse_column(fragment: str) -> Column | None:
    match = re.match(rf"\s*({IDENTIFIER})\s+(.+)", fragment, re.DOTALL)
    if not match:
        return None
    name = unquote_identifier(match.group(1))
    remainder = " ".join(match.group(2).split())
    if name.upper() in {
        "CONSTRAINT",
        "PRIMARY",
        "FOREIGN",
        "UNIQUE",
        "CHECK",
        "EXCLUDE",
        "LIKE",
    }:
        return None

    boundary = re.search(
        r"\s+(?:NOT\s+NULL|NULL|DEFAULT|CONSTRAINT|PRIMARY\s+KEY|REFERENCES|"
        r"UNIQUE|CHECK|GENERATED|COLLATE)\b",
        remainder,
        re.IGNORECASE,
    )
    data_type = remainder[: boundary.start()] if boundary else remainder
    default_match = re.search(
        r"\bDEFAULT\s+(.+?)(?=\s+(?:NOT\s+NULL|NULL|CONSTRAINT|PRIMARY\s+KEY|"
        r"REFERENCES|UNIQUE|CHECK|GENERATED|COLLATE)\b|$)",
        remainder,
        re.IGNORECASE,
    )
    return Column(
        name=name,
        data_type=data_type,
        nullable=not bool(re.search(r"\bNOT\s+NULL\b", remainder, re.IGNORECASE)),
        default=default_match.group(1).strip() if default_match else None,
        primary_key=bool(re.search(r"\bPRIMARY\s+KEY\b", remainder, re.IGNORECASE)),
        foreign_key=bool(re.search(r"\bREFERENCES\b", remainder, re.IGNORECASE)),
    )


def parse_schema(path: Path) -> list[Table]:
    sql = strip_comments(path.read_text(encoding="utf-8"))
    tables: list[Table] = []
    create_pattern = re.compile(
        rf"\bCREATE\s+(?:UNLOGGED\s+)?TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?"
        rf"({IDENTIFIER})\s*\(",
        re.IGNORECASE,
    )
    for match in create_pattern.finditer(sql):
        open_parenthesis = match.end() - 1
        close_parenthesis = find_closing_parenthesis(sql, open_parenthesis)
        table = Table(name=unquote_identifier(match.group(1)), source=path.name)
        for fragment in split_top_level(sql[open_parenthesis + 1 : close_parenthesis]):
            primary_match = re.search(
                r"(?:CONSTRAINT\s+\S+\s+)?PRIMARY\s+KEY\s*\((.*?)\)",
                fragment,
                re.IGNORECASE | re.DOTALL,
            )
            if primary_match:
                table.primary_key = parse_identifiers(primary_match.group(1))
                continue
            foreign_key = parse_foreign_key(fragment)
            if foreign_key:
                table.foreign_keys.append(foreign_key)
                continue
            column = parse_column(fragment)
            if column:
                table.columns.append(column)

        inline_primary_key = tuple(column.name for column in table.columns if column.primary_key)
        if not table.primary_key and inline_primary_key:
            table.primary_key = inline_primary_key
        tables.append(table)

    by_name = {table.name: table for table in tables}
    alter_pattern = re.compile(
        rf"\bALTER\s+TABLE\s+(?:ONLY\s+)?({IDENTIFIER})\s+"
        rf"ADD\s+(?:CONSTRAINT\s+{IDENTIFIER}\s+)?FOREIGN\s+KEY\s*"
        rf"\((.*?)\)\s+REFERENCES\s+({IDENTIFIER})\s*\((.*?)\)",
        re.IGNORECASE | re.DOTALL,
    )
    for match in alter_pattern.finditer(sql):
        table = by_name.get(unquote_identifier(match.group(1)))
        if not table:
            continue
        foreign_key = ForeignKey(
            columns=parse_identifiers(match.group(2)),
            target_table=unquote_identifier(match.group(3)),
            target_columns=parse_identifiers(match.group(4)),
        )
        if foreign_key not in table.foreign_keys:
            table.foreign_keys.append(foreign_key)

    for table in tables:
        primary_columns = set(table.primary_key)
        foreign_columns = {
            column for foreign_key in table.foreign_keys for column in foreign_key.columns
        }
        for column in table.columns:
            column.primary_key = column.name in primary_columns
            column.foreign_key = column.name in foreign_columns
            if column.primary_key:
                column.nullable = False
    return tables


def serialize(tables: list[Table], schema_paths: tuple[Path, ...]) -> dict[str, object]:
    relationships = []
    for table in tables:
        for foreign_key in table.foreign_keys:
            relationships.append(
                {
                    "source": table.name,
                    "columns": foreign_key.columns,
                    "target": foreign_key.target_table,
                    "targetColumns": foreign_key.target_columns,
                }
            )
    return {
        "name": MODEL_NAME,
        "tables": [
            {
                "name": table.name,
                "source": table.source,
                "primaryKey": table.primary_key,
                "columns": [
                    {
                        "name": column.name,
                        "type": column.data_type,
                        "nullable": column.nullable,
                        "default": column.default,
                        "primaryKey": column.primary_key,
                        "foreignKey": column.foreign_key,
                    }
                    for column in table.columns
                ],
                "foreignKeys": [
                    {
                        "columns": foreign_key.columns,
                        "target": foreign_key.target_table,
                        "targetColumns": foreign_key.target_columns,
                    }
                    for foreign_key in table.foreign_keys
                ],
            }
            for table in tables
        ],
        "relationships": relationships,
        "sources": [display_path(path) for path in schema_paths],
        "sourceConfig": display_path(SQLC_CONFIG),
    }


def _render_legacy_html(hatchet_oss_model: dict[str, object]) -> str:
    payload = json.dumps(hatchet_oss_model, separators=(",", ":")).replace("</", "<\\/")
    table_count = len(hatchet_oss_model["tables"])
    relationship_count = len(hatchet_oss_model["relationships"])
    return f"""<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>hatchet-oss database model</title>
<style>
:root {{
  color-scheme: dark;
  --bg: #0b1020; --panel: #11182b; --panel2: #182238; --line: #34435f;
  --text: #e7edf7; --muted: #8fa0ba; --accent: #6ee7b7; --pk: #fbbf24; --fk: #60a5fa;
}}
* {{ box-sizing: border-box; }}
html, body {{ height: 100%; margin: 0; overflow: hidden; }}
body {{ background: var(--bg); color: var(--text); font: 13px Inter, ui-sans-serif, system-ui, sans-serif; }}
button, input, select {{ color: inherit; font: inherit; }}
header {{
  height: 58px; display: flex; align-items: center; gap: 10px; padding: 9px 14px;
  background: #0d1425; border-bottom: 1px solid #27334b;
}}
h1 {{ font-size: 16px; margin: 0 10px 0 0; white-space: nowrap; }}
.search {{ width: min(380px, 30vw); padding: 8px 10px; border: 1px solid #35435c; border-radius: 7px; background: #111a2d; }}
select, button {{ border: 1px solid #35435c; border-radius: 7px; background: #172136; padding: 7px 10px; }}
button {{ cursor: pointer; }}
button:hover {{ background: #21304b; }}
.stats {{ margin-left: auto; color: var(--muted); white-space: nowrap; }}
.shell {{ height: calc(100% - 58px); display: grid; grid-template-columns: 260px 1fr 330px; }}
aside {{ background: var(--panel); min-width: 0; overflow: hidden; }}
.left {{ border-right: 1px solid #27334b; display: flex; flex-direction: column; }}
.right {{ border-left: 1px solid #27334b; overflow: auto; padding: 16px; }}
.source-note {{ padding: 11px 13px; color: var(--muted); border-bottom: 1px solid #27334b; line-height: 1.45; }}
.table-list {{ overflow: auto; padding: 7px; }}
.table-link {{ display: block; width: 100%; padding: 7px 9px; text-align: left; border: 0; background: none; color: var(--muted); }}
.table-link:hover, .table-link.active {{ color: var(--text); background: #1b2740; }}
.table-link small {{ float: right; color: #63728c; }}
#viewport {{ position: relative; overflow: hidden; cursor: grab; background-image: radial-gradient(#26334b 1px, transparent 1px); background-size: 24px 24px; }}
#viewport.dragging {{ cursor: grabbing; }}
#world {{ position: absolute; transform-origin: 0 0; }}
#edges {{ position: absolute; inset: 0; overflow: visible; pointer-events: none; }}
.edge {{ fill: none; stroke: #50698f; stroke-width: 1.5; opacity: .55; marker-end: url(#arrow); }}
.edge.active {{ stroke: var(--accent); stroke-width: 2.5; opacity: 1; }}
.group-label {{ position: absolute; color: #7787a1; font-size: 18px; font-weight: 700; letter-spacing: .04em; }}
.node {{
  position: absolute; width: 270px; border: 1px solid #3a4964; border-radius: 8px;
  background: #121b2e; box-shadow: 0 5px 15px #0005; overflow: hidden; cursor: pointer;
}}
.node:hover, .node.selected {{ border-color: var(--accent); box-shadow: 0 0 0 1px var(--accent), 0 6px 20px #0008; }}
.node.hidden, .edge.hidden, .group-label.hidden {{ display: none; }}
.node-title {{ padding: 8px 10px; background: var(--panel2); font-weight: 700; overflow: hidden; text-overflow: ellipsis; }}
.node-source {{ float: right; color: #71819c; font-size: 10px; font-weight: 500; }}
.column {{ display: grid; grid-template-columns: 19px minmax(0,1fr) auto; gap: 5px; padding: 3px 8px; border-top: 1px solid #1f2a40; font-size: 11px; }}
.key {{ color: var(--pk); font-weight: 700; }}
.fk {{ color: var(--fk); }}
.column-name {{ overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }}
.column-type {{ color: var(--muted); max-width: 110px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }}
.more {{ padding: 4px 9px 6px; color: #71819c; font-size: 11px; }}
.empty {{ color: var(--muted); padding: 12px; }}
.right h2 {{ margin: 0 0 4px; font-size: 18px; overflow-wrap: anywhere; }}
.badge {{ display: inline-block; margin: 3px 4px 8px 0; padding: 2px 6px; border-radius: 10px; background: #25314a; color: #aab6c9; font-size: 10px; }}
.detail-source {{ color: var(--muted); margin-bottom: 14px; }}
.detail-column {{ padding: 8px 0; border-top: 1px solid #27334b; }}
.detail-column strong {{ display: block; overflow-wrap: anywhere; }}
.detail-type {{ color: #a8b6ca; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 11px; overflow-wrap: anywhere; }}
.detail-meta {{ color: var(--muted); font-size: 11px; margin-top: 3px; overflow-wrap: anywhere; }}
.relations {{ margin-top: 16px; }}
.relation {{ padding: 6px 0; color: #a8b6ca; overflow-wrap: anywhere; }}
.relation button {{ border: 0; background: none; color: var(--fk); padding: 0; }}
@media (max-width: 1000px) {{ .shell {{ grid-template-columns: 220px 1fr; }} .right {{ display: none; }} }}
@media (max-width: 680px) {{ .shell {{ grid-template-columns: 1fr; }} .left {{ display: none; }} .stats {{ display: none; }} }}
</style>
</head>
<body>
<header>
  <h1>hatchet-oss database model</h1>
  <input id="search" class="search" type="search" placeholder="Search tables, columns, or types" aria-label="Search model">
  <select id="source" aria-label="Filter schema source"><option value="">All schema files</option></select>
  <button id="zoomOut" title="Zoom out">−</button>
  <button id="zoomIn" title="Zoom in">+</button>
  <button id="fit">Fit</button>
  <span class="stats">{table_count} tables · {relationship_count} foreign keys</span>
</header>
<main class="shell">
  <aside class="left">
    <div class="source-note">Generated from the canonical OSS SQL schema snapshots listed in sqlc.yaml. No database connection is used.</div>
    <nav id="tableList" class="table-list" aria-label="Tables"></nav>
  </aside>
  <section id="viewport" aria-label="Entity relationship diagram">
    <div id="world"><svg id="edges"><defs><marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="5" markerHeight="5" orient="auto-start-reverse"><path d="M 0 0 L 10 5 L 0 10 z" fill="#50698f"/></marker></defs></svg></div>
  </section>
  <aside id="details" class="right"><div class="empty">Select a table to inspect all columns and relationships.</div></aside>
</main>
<script>
const hatchetOssModel = {payload};
const state = {{ scale: 0.55, x: 20, y: 20, selected: null, visible: new Set() }};
const positions = new Map();
const nodes = new Map();
const edgeElements = [];
const world = document.getElementById('world');
const viewport = document.getElementById('viewport');
const list = document.getElementById('tableList');
const details = document.getElementById('details');
const search = document.getElementById('search');
const sourceFilter = document.getElementById('source');

for (const source of [...new Set(hatchetOssModel.tables.map(table => table.source))]) {{
  const option = document.createElement('option');
  option.value = source;
  option.textContent = source;
  sourceFilter.append(option);
}}

function layout() {{
  const groups = [...new Set(hatchetOssModel.tables.map(table => table.source))];
  let groupY = 45;
  let worldWidth = 0;
  for (const group of groups) {{
    const tables = hatchetOssModel.tables.filter(table => table.source === group);
    const columns = Math.min(5, Math.max(2, Math.ceil(Math.sqrt(tables.length))));
    const heights = Array(columns).fill(groupY + 38);
    const label = document.createElement('div');
    label.className = 'group-label';
    label.dataset.source = group;
    label.style.left = '16px';
    label.style.top = `${{groupY}}px`;
    label.textContent = group;
    world.append(label);
    for (const table of tables) {{
      const column = heights.indexOf(Math.min(...heights));
      const shown = Math.min(table.columns.length, 9);
      const height = 37 + shown * 18 + (table.columns.length > shown ? 24 : 0);
      positions.set(table.name, {{ x: 20 + column * 300, y: heights[column], width: 270, height }});
      heights[column] += height + 28;
    }}
    groupY = Math.max(...heights) + 70;
    worldWidth = Math.max(worldWidth, columns * 300);
  }}
  world.dataset.width = worldWidth;
  world.dataset.height = groupY;
}}

function keyLabel(column) {{
  if (column.primaryKey && column.foreignKey) return 'PK/FK';
  if (column.primaryKey) return 'PK';
  if (column.foreignKey) return 'FK';
  return '';
}}

function createNodes() {{
  for (const table of hatchetOssModel.tables) {{
    const position = positions.get(table.name);
    const node = document.createElement('article');
    node.className = 'node';
    node.dataset.name = table.name;
    node.dataset.source = table.source;
    node.style.cssText = `left:${{position.x}}px;top:${{position.y}}px;height:${{position.height}}px`;
    const title = document.createElement('div');
    title.className = 'node-title';
    title.textContent = table.name;
    const source = document.createElement('span');
    source.className = 'node-source';
    source.textContent = table.source.replace('.sql', '');
    title.append(source);
    node.append(title);
    for (const column of table.columns.slice(0, 9)) {{
      const row = document.createElement('div');
      row.className = 'column';
      const key = document.createElement('span');
      key.className = column.primaryKey ? 'key' : column.foreignKey ? 'fk' : '';
      key.textContent = keyLabel(column);
      const name = document.createElement('span');
      name.className = 'column-name';
      name.textContent = column.name;
      const type = document.createElement('span');
      type.className = 'column-type';
      type.textContent = column.type + (column.nullable ? '?' : '');
      row.append(key, name, type);
      node.append(row);
    }}
    if (table.columns.length > 9) {{
      const more = document.createElement('div');
      more.className = 'more';
      more.textContent = `+ ${{table.columns.length - 9}} more columns`;
      node.append(more);
    }}
    node.addEventListener('click', event => {{
      event.stopPropagation();
      selectTable(table.name, false);
    }});
    world.append(node);
    nodes.set(table.name, node);
  }}
}}

function createEdges() {{
  const svg = document.getElementById('edges');
  svg.setAttribute('width', world.dataset.width);
  svg.setAttribute('height', world.dataset.height);
  for (const relationship of hatchetOssModel.relationships) {{
    const from = positions.get(relationship.source);
    const to = positions.get(relationship.target);
    if (!from || !to) continue;
    const startX = from.x + from.width / 2;
    const startY = from.y + from.height / 2;
    const endX = to.x + to.width / 2;
    const endY = to.y + to.height / 2;
    const bend = Math.max(50, Math.abs(endX - startX) * 0.45);
    const direction = endX >= startX ? 1 : -1;
    const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
    path.setAttribute('d', `M ${{startX}} ${{startY}} C ${{startX + bend * direction}} ${{startY}}, ${{endX - bend * direction}} ${{endY}}, ${{endX}} ${{endY}}`);
    path.setAttribute('class', 'edge');
    path.dataset.source = relationship.source;
    path.dataset.target = relationship.target;
    svg.append(path);
    edgeElements.push(path);
  }}
}}

function renderList() {{
  list.replaceChildren();
  const visibleTables = hatchetOssModel.tables.filter(table => state.visible.has(table.name));
  for (const table of visibleTables) {{
    const button = document.createElement('button');
    button.className = 'table-link' + (state.selected === table.name ? ' active' : '');
    button.textContent = table.name;
    const count = document.createElement('small');
    count.textContent = table.columns.length;
    button.append(count);
    button.addEventListener('click', () => selectTable(table.name, true));
    list.append(button);
  }}
  if (!visibleTables.length) {{
    const empty = document.createElement('div');
    empty.className = 'empty';
    empty.textContent = 'No matching tables.';
    list.append(empty);
  }}
}}

function renderDetails(table) {{
  details.replaceChildren();
  const heading = document.createElement('h2');
  heading.textContent = table.name;
  const source = document.createElement('div');
  source.className = 'detail-source';
  source.textContent = table.source;
  details.append(heading, source);
  for (const column of table.columns) {{
    const row = document.createElement('div');
    row.className = 'detail-column';
    const name = document.createElement('strong');
    name.textContent = column.name;
    const type = document.createElement('div');
    type.className = 'detail-type';
    type.textContent = column.type;
    const meta = document.createElement('div');
    meta.className = 'detail-meta';
    const labels = [column.nullable ? 'nullable' : 'not null'];
    if (column.primaryKey) labels.push('primary key');
    if (column.foreignKey) labels.push('foreign key');
    if (column.default !== null) labels.push(`default: ${{column.default}}`);
    meta.textContent = labels.join(' · ');
    row.append(name, type, meta);
    details.append(row);
  }}
  const outgoing = hatchetOssModel.relationships.filter(item => item.source === table.name);
  const incoming = hatchetOssModel.relationships.filter(item => item.target === table.name);
  const section = document.createElement('div');
  section.className = 'relations';
  const title = document.createElement('strong');
  title.textContent = `Relationships (${{outgoing.length + incoming.length}})`;
  section.append(title);
  for (const relationship of outgoing) {{
    section.append(relationRow(`${{relationship.columns.join(', ')}} → `, relationship.target, relationship.targetColumns.join(', ')));
  }}
  for (const relationship of incoming) {{
    section.append(relationRow('← ', relationship.source, relationship.columns.join(', ')));
  }}
  if (!outgoing.length && !incoming.length) {{
    const empty = document.createElement('div');
    empty.className = 'empty';
    empty.textContent = 'No explicit foreign keys.';
    section.append(empty);
  }}
  details.append(section);
}}

function relationRow(prefix, target, suffix) {{
  const row = document.createElement('div');
  row.className = 'relation';
  row.append(document.createTextNode(prefix));
  const button = document.createElement('button');
  button.textContent = target;
  button.addEventListener('click', () => selectTable(target, true));
  row.append(button, document.createTextNode(` (${{suffix}})`));
  return row;
}}

function selectTable(name, center) {{
  state.selected = name;
  for (const [nodeName, node] of nodes) node.classList.toggle('selected', nodeName === name);
  for (const edge of edgeElements) edge.classList.toggle('active', edge.dataset.source === name || edge.dataset.target === name);
  const table = hatchetOssModel.tables.find(item => item.name === name);
  if (table) renderDetails(table);
  renderList();
  if (center && positions.has(name)) centerTable(name);
}}

function applyFilter() {{
  const query = search.value.trim().toLowerCase();
  const source = sourceFilter.value;
  state.visible.clear();
  for (const table of hatchetOssModel.tables) {{
    const haystack = [table.name, table.source, ...table.columns.flatMap(column => [column.name, column.type])].join(' ').toLowerCase();
    const visible = (!source || table.source === source) && (!query || haystack.includes(query));
    nodes.get(table.name).classList.toggle('hidden', !visible);
    if (visible) state.visible.add(table.name);
  }}
  for (const edge of edgeElements) {{
    edge.classList.toggle('hidden', !state.visible.has(edge.dataset.source) || !state.visible.has(edge.dataset.target));
  }}
  for (const label of world.querySelectorAll('.group-label')) {{
    const anyVisible = hatchetOssModel.tables.some(table => table.source === label.dataset.source && state.visible.has(table.name));
    label.classList.toggle('hidden', !anyVisible);
  }}
  renderList();
}}

function updateTransform() {{
  world.style.transform = `translate(${{state.x}}px, ${{state.y}}px) scale(${{state.scale}})`;
}}

function setZoom(next, anchorX = viewport.clientWidth / 2, anchorY = viewport.clientHeight / 2) {{
  const old = state.scale;
  state.scale = Math.max(0.12, Math.min(1.8, next));
  state.x = anchorX - (anchorX - state.x) * state.scale / old;
  state.y = anchorY - (anchorY - state.y) * state.scale / old;
  updateTransform();
}}

function fit() {{
  const visible = [...state.visible].map(name => positions.get(name)).filter(Boolean);
  if (!visible.length) return;
  const minX = Math.min(...visible.map(item => item.x)) - 25;
  const minY = Math.min(...visible.map(item => item.y)) - 25;
  const maxX = Math.max(...visible.map(item => item.x + item.width)) + 25;
  const maxY = Math.max(...visible.map(item => item.y + item.height)) + 25;
  state.scale = Math.max(0.12, Math.min(1.2, Math.min(viewport.clientWidth / (maxX - minX), viewport.clientHeight / (maxY - minY))));
  state.x = (viewport.clientWidth - (maxX - minX) * state.scale) / 2 - minX * state.scale;
  state.y = (viewport.clientHeight - (maxY - minY) * state.scale) / 2 - minY * state.scale;
  updateTransform();
}}

function centerTable(name) {{
  const position = positions.get(name);
  state.x = viewport.clientWidth / 2 - (position.x + position.width / 2) * state.scale;
  state.y = viewport.clientHeight / 2 - (position.y + position.height / 2) * state.scale;
  updateTransform();
}}

let drag = null;
viewport.addEventListener('pointerdown', event => {{
  if (event.target.closest('.node')) return;
  drag = {{ x: event.clientX, y: event.clientY, originX: state.x, originY: state.y }};
  viewport.setPointerCapture(event.pointerId);
  viewport.classList.add('dragging');
}});
viewport.addEventListener('pointermove', event => {{
  if (!drag) return;
  state.x = drag.originX + event.clientX - drag.x;
  state.y = drag.originY + event.clientY - drag.y;
  updateTransform();
}});
viewport.addEventListener('pointerup', () => {{ drag = null; viewport.classList.remove('dragging'); }});
viewport.addEventListener('wheel', event => {{
  event.preventDefault();
  const rect = viewport.getBoundingClientRect();
  setZoom(state.scale * (event.deltaY > 0 ? 0.88 : 1.14), event.clientX - rect.left, event.clientY - rect.top);
}}, {{ passive: false }});
viewport.addEventListener('click', () => selectTable(null, false));
search.addEventListener('input', applyFilter);
sourceFilter.addEventListener('change', applyFilter);
document.getElementById('zoomIn').addEventListener('click', () => setZoom(state.scale * 1.2));
document.getElementById('zoomOut').addEventListener('click', () => setZoom(state.scale / 1.2));
document.getElementById('fit').addEventListener('click', fit);

layout();
createNodes();
createEdges();
applyFilter();
requestAnimationFrame(fit);
</script>
</body>
</html>
"""


def render_html(hatchet_oss_model: dict[str, object]) -> str:
    payload = json.dumps(hatchet_oss_model, separators=(",", ":")).replace(
        "</", "<\\/"
    )
    template = HTML_TEMPLATE.read_text(encoding="utf-8")
    marker = "__HATCHET_OSS_MODEL__"
    if template.count(marker) != 1:
        raise ValueError(f"expected exactly one {marker} marker in {HTML_TEMPLATE}")
    return template.replace(marker, payload)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Generate a standalone HTML visualization of the hatchet-oss model."
    )
    parser.add_argument(
        "--output",
        type=Path,
        default=DEFAULT_OUTPUT,
        help=f"HTML output path (default: {DEFAULT_OUTPUT.relative_to(ROOT)})",
    )
    parser.add_argument(
        "--schema",
        type=Path,
        action="append",
        dest="schemas",
        help="SQL schema path; repeat to override discovery from the OSS sqlc config",
    )
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    selected_schemas = args.schemas or discover_oss_schema_paths()
    schema_paths = tuple(
        path if path.is_absolute() else ROOT / path
        for path in selected_schemas
    )
    missing = [str(path) for path in schema_paths if not path.is_file()]
    if missing:
        raise SystemExit(f"schema file not found: {', '.join(missing)}")

    tables = [
        table
        for schema_path in schema_paths
        for table in parse_schema(schema_path)
    ]
    duplicate_names = sorted(
        name for name in {table.name for table in tables} if sum(table.name == name for table in tables) > 1
    )
    if duplicate_names:
        raise SystemExit(f"duplicate table names across schemas: {', '.join(duplicate_names)}")
    if not tables:
        raise SystemExit("no CREATE TABLE statements found in the selected schemas")

    hatchet_oss_model = serialize(tables, schema_paths)
    output = args.output if args.output.is_absolute() else ROOT / args.output
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(render_html(hatchet_oss_model), encoding="utf-8")
    print(
        f"Wrote {display_path(output)} with "
        f"{len(hatchet_oss_model['tables'])} tables and "
        f"{len(hatchet_oss_model['relationships'])} foreign keys."
    )


if __name__ == "__main__":
    main()
