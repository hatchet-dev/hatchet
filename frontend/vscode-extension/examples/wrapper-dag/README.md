# Wrapper DAG examples

Each file defines a reusable **workflow wrapper** (a factory function marked
`@hatchet-workflow`) and then **uses** it — the task graph is attached at the
usage site, and the DAG renders there. All four define the same diamond:

```
        start
        /   \
   branch-a branch-b
        \   /
         join
```

| File          | Language   | Wrapper marker          | DAG renders on |
| ------------- | ---------- | ----------------------- | -------------- |
| `workflow.ts` | TypeScript | `@hatchet-workflow` JSDoc tag | `ordersDag` |
| `workflow.py` | Python     | `# @hatchet-workflow`   | `orders_dag`   |
| `workflow.go` | Go         | `// @hatchet-workflow` (file is `//go:build ignore`) | `ordersDag` |
| `workflow.rb` | Ruby       | `# @hatchet-workflow`   | `orders_dag`   |

The TypeScript factory body additionally uses a **dynamic name** (`stub.name`)
with explicit generics — the original case that hid the workflow from the
visualizer.

## How the wrapper feature works

- A function marked `@hatchet-workflow` is treated as a workflow factory.
- A usage of that factory (`const ordersDag = createWorkflowBuilder(...)`,
  `orders_dag = create_workflow(...)`, etc.) is treated as a workflow
  declaration, so tasks attached to the returned value form the DAG.
- The DAG label comes from a `name` argument in the usage call when present
  (here `"orders-dag"`), otherwise the usage variable name.

> Note: for Python/Go/Ruby the wrapper and its usage must be in the **same
> file** (these parsers scan a single file). The TypeScript wrapper is resolved
> across the workspace — see [`cross-file/`](cross-file/), where the wrapper
> (`builder.ts`) and its usage (`orders-dag.ts`) live in separate files and the
> DAG still renders on the usage.

## How to visualize

1. Open the `frontend/vscode-extension` folder in VSCode.
2. Press **F5** (Run → Start Debugging → "Run Extension (wrapper-dag examples)").
   It builds the extension and opens this folder in an Extension Development Host.
3. Open any `workflow.*` file and click the **"Visualize Hatchet DAG"** CodeLens
   above the usage variable.

These are illustrative — not wired to a running Hatchet instance, not meant to
compile/run.
