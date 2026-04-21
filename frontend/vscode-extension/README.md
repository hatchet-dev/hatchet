# Hatchet VSCode Extension

This is an extension for [Hatchet](https://github.com/hatchet-dev/hatchet) which allows visualization of Hatchet [DAGs](https://docs.hatchet.run/v1/patterns/directed-acyclic-graphs). This works with Hatchet's Python, Typescript, Go and Ruby SDKs.

## Usage

After installation, each Hatchet DAG definition will show a button `Show Hatchet DAG`:

![VSCode Button](https://raw.githubusercontent.com/hatchet-dev/hatchet/main/frontend/vscode-extension/assets/vscode_button.png)

This will then open a webview showing the DAG definition. As you update your definition, the webview will automatically stay up to date:

![Webview](https://raw.githubusercontent.com/hatchet-dev/hatchet/main/frontend/vscode-extension/assets/vscode_dag.png)

## Custom workflow wrappers

If your codebase wraps the Hatchet workflow builder, annotate your factory function with `@hatchet-workflow` so the extension can detect variables it creates:

```typescript
/**
 * @hatchet-workflow
 * @hatchet-task-method task
 * @hatchet-task-parents parents
 */
export function createWorkflowBuilder(options) { ... }
```

The extension scans your workspace for annotated functions on startup and places a **Show Hatchet DAG** CodeLens above every variable created by one. Two optional tags let you match your wrapper's API:

| Tag | Default | Purpose |
|---|---|---|
| `@hatchet-task-method` | `task` | Method name used to register tasks |
| `@hatchet-task-parents` | `parents` | Option property that lists parent tasks |

Both `wf.task('name', { parents })` and `wf.task({ name, parents })` call signatures are supported automatically.

## Reporting issues

You can report issues in our [Github issues](https://github.com/hatchet-dev/hatchet/issues).

## Changelog

### 0.2.1

- **Fix memory leak** — workspace annotation scanning now reads files via `vscode.workspace.fs.readFile` instead of `openTextDocument`, preventing Monaco text models from being created for every `.ts`/`.tsx` file in the workspace.

### 0.2.0

- **Custom wrapper support** — annotate any factory function with `@hatchet-workflow` to get DAG visualization for variables it creates, with no other config required. Use `@hatchet-task-method` and `@hatchet-task-parents` to match your wrapper's API when it differs from the Hatchet defaults.
- Both positional-name (`wf.task('name', opts)`) and options-object (`wf.task({ name, parents })`) task call signatures are now supported.
- Cross-file task resolution via LSP now honours custom method and parents property names.

### 0.1.2

- Initial release with DAG visualization for TypeScript, Python, Go, and Ruby workflows.
