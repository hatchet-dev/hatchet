# Testing the Hatchet VS Code Extension Locally

## Prerequisites

- Node.js 20+
- pnpm (`npm install -g pnpm` or `corepack enable`)
- VS Code

## One-time setup

```bash
cd frontend/vscode-extension
pnpm install
```

## Running the extension in a development host

1. Open the `frontend/vscode-extension` folder in VS Code:

   ```bash
   code frontend/vscode-extension
   ```

2. Press **F5** (or go to **Run → Start Debugging → "Run Extension"**).

   VS Code will automatically run `pnpm build` and then open a new **Extension Development Host** window with the extension loaded.

3. In the Extension Development Host window, open any Hatchet workflow file. Good examples are in this repo:

   - `examples/typescript/dag/workflow.ts`
   - `examples/python/` (any workflow file)
   - `examples/go/` (any workflow file)

4. A **"$(graph) Show Hatchet DAG"** CodeLens should appear above each workflow declaration. Click it to open the DAG panel.

## Watch mode (rebuild on save)

In one terminal:

```bash
cd frontend/vscode-extension
pnpm watch
```

Then press **F5** in VS Code as above. After any source change, press **Ctrl+Shift+F5** (Restart) in the debug toolbar to reload with the latest build.

## Typecheck without running

```bash
pnpm typecheck
```

## Testing cross-file LSP analysis

The LSP phase requires a language server to be running. For TypeScript, the built-in TypeScript language server is always available. To exercise cross-file DAG resolution:

1. Create a workflow declaration in one file and add tasks in a second file that imports and calls methods on the workflow variable.
2. Open the file with the workflow declaration and click the CodeLens.
3. The panel should briefly show "Analyzing…" then resolve tasks from both files.

To test the fallback path (language server unavailable), disable the TypeScript extension and reload the window — the panel will show with a warning banner and fall back to single-file parsing.
