# Hatchet VSCode extension — developer guide

This extension renders a **DAG visualizer** for Hatchet workflows. When you open
a workflow file (TypeScript, Python, Go, or Ruby), it adds a **CodeLens**
(`$(graph) Show Hatchet DAG — <name>`) above each workflow declaration; clicking
it opens a webview panel that draws the task graph and live-updates as you edit.

This guide covers how the pieces fit together, the patterns it recognizes in
each language, and — most importantly — **how to test a change**.

---

## Quick start

```bash
cd frontend/vscode-extension
pnpm install
pnpm run typecheck    # type-check (see "Gotcha: transpileOnly" below — the build does NOT)
pnpm run build        # bundle to dist/extension.js (webpack, dev mode)
```

Then press **F5** in VSCode to launch it (see [Testing](#testing)).

This package uses **pnpm**. There is no automated test suite yet; see
[Testing](#testing) for the manual + harness workflow.

---

## Architecture

The flow from "user opens a file" to "DAG appears":

```
                      ┌─────────────────────────────────────────────┐
   file opened/edited │                                             │
        │             │   WorkflowAnnotationCache (TS only)         │
        ▼             │   scans workspace *.ts for @hatchet-workflow │
  CodeLensProvider ◄──┤   factory functions, keeps them cached      │
   (provideCodeLenses)│                                             │
        │             └─────────────────────────────────────────────┘
        │  detectWorkflowDeclarations(text, languageId, …)   ← fast "pass 1" scan
        │     → per-language detector returns WorkflowDeclaration[]
        ▼
   CodeLens "Show Hatchet DAG — <name>"   (command: hatchet.showDag)
        │  user clicks
        ▼
   command hatchet.showDag (extension.ts)
        │  computeFallbackWorkflow(...) → single-file ParsedWorkflow
        ▼
   DagPanel.createOrShow(...)
        │  LspAnalyzer.analyzeWorkflow(...)   ← "pass 2": resolve tasks,
        │     uses the language server to find cross-file task references,
        │     falls back to the single-file parse if LSP is unavailable
        ▼
   webview (dag-visualizer) draws nodes + edges
```

Two-phase parsing is the key idea:

- **Pass 1 — detect (fast):** `detect<Lang>WorkflowDeclarations(text)` finds just
  the workflow *variables* (name + line). Cheap enough to run on every keystroke
  for CodeLens placement.
- **Pass 2 — parse (full):** `parse<Lang>Workflows(text)` / `LspAnalyzer` resolve
  the **tasks** and their **parent edges** that make up the DAG shape.

### File map

| Path | Responsibility |
| --- | --- |
| `src/extension.ts` | Activation, registers the CodeLens provider for each language, the `hatchet.showDag` command, and edit/active-editor listeners. |
| `src/providers/codelens-provider.ts` | `looksLikeHatchetDocument` pre-filter, then dispatches to the per-language detect/parse functions. The single switchboard over `languageId`. |
| `src/parser/workflow-parser.ts` | **TypeScript** parser (uses the TypeScript compiler API / AST). Handles generics, dynamic names, and the `@hatchet-workflow` wrapper path. |
| `src/parser/python-parser.ts`, `go-parser.ts`, `ruby-parser.ts` | **Regex/line-based** parsers for the other languages. |
| `src/parser/jsdoc-annotations.ts` | Scans TS source for `@hatchet-workflow` JSDoc tags (factory functions). |
| `src/parser/wrapper-annotations.ts` | Shared helpers for the comment-marker wrapper feature in the Python/Go/Ruby parsers. |
| `src/analysis/annotation-cache.ts` | Workspace-wide cache of TS `@hatchet-workflow` factories, kept fresh by a file watcher. |
| `src/analysis/lsp-analyzer.ts` | Pass 2: queries the language server for cross-file task references and extracts tasks from each reference site. |
| `src/panel/dag-panel.ts` | Owns the webview panel lifecycle and debounced updates. |
| `src/dag-visualizer/`, `src/webview/` | The React webview that lays out and draws the graph. |
| `src/utils/workspace.ts` | Workspace-boundary helpers (e.g. ignore `node_modules`). |

---

## What each language parser recognizes

All four recognize three things: the **workflow** declaration, **task**
declarations on that workflow variable, and each task's **parents**.

### Workflow name

The workflow name (the DAG label) may be a **string literal** or a **dynamic
expression** — for a non-literal like `stub.name`, the expression text is used
as the label so the workflow still renders.

| Language | Workflow declaration |
| --- | --- |
| TypeScript | `const wf = hatchet.workflow<...>({ name: "x" \| stub.name })` |
| Python | `wf = hatchet.workflow(name="x" \| stub.name)` |
| Go | `wf := client.NewWorkflow("x" \| stub.Name)` |
| Ruby | `wf = hatchet.workflow(name: "x" \| stub.name)` |

### Tasks and parents

| Language | Task | Parents |
| --- | --- | --- |
| TypeScript | `wf.task({ name, parents })` or `wf.task("name", { parents })` | `parents: [a, b]` |
| Python | `@wf.task(...)` above `def step(...)` | `parents=[a, b]` |
| Go | `s := wf.NewTask("name", ...)` | `hatchet.WithParents(a, b)` |
| Ruby | `s = wf.task(:name, ...)` | `parents: [a, b]` |

### The `@hatchet-workflow` wrapper feature

A factory function can be **marked as a workflow wrapper**, so a *usage* of it is
treated as a workflow and the tasks attached to the returned value form the DAG:

```ts
/** @hatchet-workflow */
function createWorkflowBuilder(stub) { return hatchet.workflow({ name: stub.name }); }

const ordersDag = createWorkflowBuilder({ name: "orders-dag" });  // ← DAG renders here
const start = ordersDag.task({ name: "start" });
```

- **TypeScript:** marker is the `@hatchet-workflow` **JSDoc tag**; wrappers are
  resolved **across the workspace** (via `WorkflowAnnotationCache`).
- **Python/Go/Ruby:** marker is a `@hatchet-workflow` **comment** on/above the
  function (`# @hatchet-workflow` / `// @hatchet-workflow`); wrappers are resolved
  **within the same file only** (the regex parsers scan a single file).

Runnable examples for all four live in [`examples/wrapper-dag/`](examples/wrapper-dag/).

---

## Testing

### 1. Manually, in an Extension Development Host (the real thing)

1. Open the `frontend/vscode-extension` folder in VSCode.
2. Press **F5** → **"Run Extension (wrapper-dag examples)"**. The launch config
   (`.vscode/launch.json`) runs the `build` task first and opens
   `examples/wrapper-dag/` in a second VSCode window with the extension loaded.
3. Open any `examples/wrapper-dag/workflow.*` file. A `Show Hatchet DAG` CodeLens
   should appear above the workflow variable. Click it → the DAG panel opens.
4. Edit a task (add one, change a `parents` list) and watch the panel update.

To reload after changing extension code: rebuild (`pnpm run build`, or run
`pnpm run watch` in a terminal) and use **Developer: Reload Window** in the dev
host. The `watch` script keeps `dist/` rebuilding on save.

### 2. Parser logic, with a throwaway harness (fast, no deps)

The parsers are plain functions, so you can exercise them directly without
VSCode. The TS parser imports `typescript` (a real dep) and only *types* from
`vscode`, so it runs fine under plain Node. Recipe:

```bash
cd frontend/vscode-extension
# write a quick script, e.g. /tmp/t.ts, importing from ./src/parser/...
node_modules/.bin/tsc /tmp/t.ts --module commonjs --target ES2020 \
  --moduleResolution node --esModuleInterop --skipLibCheck \
  --outDir ./__scratch && node ./__scratch/t.js && rm -rf ./__scratch
```

Example `t.ts`:

```ts
import { detectPyWorkflowDeclarations, parsePythonWorkflows } from './src/parser/python-parser';

const src = `
# @hatchet-workflow
def make(hatchet, name):
    return hatchet.workflow(name=name)

wf = make(hatchet, name="demo")

@wf.task()
def start(i, c): ...
@wf.task(parents=[start])
def end(i, c): ...
`;

console.log(detectPyWorkflowDeclarations(src));        // → [{ name: 'demo', varName: 'wf', ... }]
console.log(parsePythonWorkflows(src)[0].tasks);       // → start, end (end parented to start)
```

This is exactly how the parser changes in this folder were verified. Keep the
output directory and scratch file outside `src/` so they aren't bundled.

### 3. Static checks (treat as CI)

```bash
pnpm run typecheck    # MUST pass — the build does not type-check (see gotcha)
pnpm run build        # MUST succeed and emit dist/extension.js
```

### Want a real test suite?

There is none yet. The lightest path is **vitest** (no VSCode runtime needed for
the parsers): `pnpm add -D vitest`, add `"test": "vitest run"`, and move harness
cases into `src/parser/__tests__/*.test.ts`. The `LspAnalyzer` and panel need the
VSCode API, so those are better covered by the manual flow or
`@vscode/test-electron`.

---

## Build & packaging

- `pnpm run build` / `build:prod` — webpack bundle (dev / production).
- `pnpm run watch` — rebuild on change (use while developing in the dev host).
- `pnpm run package` — produce a `.vsix` via `vsce` (installable with
  *Extensions: Install from VSIX…*).
- `pnpm run publish` — publish via `vsce` (maintainers only).

### Gotcha: `transpileOnly`

The webpack build uses `ts-loader` with **`transpileOnly: true`**, so **type
errors do not fail the build** — they surface only at runtime (or never, if the
bad code path isn't hit). Always run `pnpm run typecheck` before trusting a
build. (A real example: an out-of-scope variable reference compiled fine and
silently broke cross-file task resolution until typecheck flagged it.)

---

## Adding support for a new pattern or language

1. Add/extend a parser in `src/parser/` exposing two functions:
   `detect<Lang>WorkflowDeclarations(text)` (pass 1) and
   `parse<Lang>Workflows(text)` (pass 2).
2. Wire the `languageId` into the switches in
   `src/providers/codelens-provider.ts` (`detectWorkflowDeclarations`,
   `parseWorkflowsForDocument`) and `looksLikeHatchetDocument`.
3. For cross-file task resolution, add an `extract<Lang>Task(...)` branch in
   `src/analysis/lsp-analyzer.ts` keyed off `languageId`.
4. Add the language to `SUPPORTED_LANGUAGES` and `activationEvents` if new.
5. Add an example under `examples/` and verify with the harness + F5.
```
