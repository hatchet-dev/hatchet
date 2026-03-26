import * as vscode from 'vscode';
import {
  scanFileForWorkflowAnnotations,
  type WorkflowFactoryAnnotation,
} from '../parser/jsdoc-annotations';

/**
 * Workspace-level cache of `@hatchet-workflow`-annotated factory functions.
 *
 * Scans all `.ts` / `.tsx` workspace files on initialization, then stays
 * up-to-date via a file-system watcher.  Fires `onDidChange` whenever the
 * set of known annotations changes so callers can refresh (e.g. CodeLens).
 */
export class WorkflowAnnotationCache {
  private readonly _annotations = new Map<string, WorkflowFactoryAnnotation>();
  private readonly _onDidChange = new vscode.EventEmitter<void>();
  readonly onDidChange: vscode.Event<void> = this._onDidChange.event;

  // ── Public API ─────────────────────────────────────────────────────────────

  /** Start the cache: scan workspace and begin watching for changes. */
  async initialize(context: vscode.ExtensionContext): Promise<void> {
    await this._scanWorkspace();

    const watcher = vscode.workspace.createFileSystemWatcher('**/*.{ts,tsx}');
    watcher.onDidChange((uri) => void this._scanUri(uri, /*remove*/ false));
    watcher.onDidCreate((uri) => void this._scanUri(uri, /*remove*/ false));
    watcher.onDidDelete((uri) => void this._scanUri(uri, /*remove*/ true));
    context.subscriptions.push(watcher, this._onDidChange);
  }

  /** Return a snapshot of all known annotations, keyed by function name. */
  getAll(): ReadonlyMap<string, WorkflowFactoryAnnotation> {
    return this._annotations;
  }

  // ── Private helpers ────────────────────────────────────────────────────────

  private async _scanWorkspace(): Promise<void> {
    const uris = await vscode.workspace.findFiles(
      '**/*.{ts,tsx}',
      '{**/node_modules/**,**/.git/**}',
    );
    await Promise.all(uris.map((uri) => this._scanUri(uri, false)));
  }

  private async _scanUri(uri: vscode.Uri, remove: boolean): Promise<void> {
    // Build a per-file key to allow removal of stale entries
    const fileKey = uri.toString();

    // Remove any previously-known annotations from this file
    const toRemove = [...this._annotations.entries()]
      .filter(([, ann]) => (ann as any)._fileKey === fileKey)
      .map(([name]) => name);

    let changed = toRemove.length > 0;
    for (const name of toRemove) this._annotations.delete(name);

    if (!remove) {
      try {
        const doc = await vscode.workspace.openTextDocument(uri);
        const text = doc.getText();
        const found = scanFileForWorkflowAnnotations(text, uri.fsPath);
        for (const ann of found) {
          // Attach a hidden file-key so we can remove stale entries later
          (ann as any)._fileKey = fileKey;
          this._annotations.set(ann.functionName, ann);
          changed = true;
        }
      } catch {
        // File unreadable — silently skip
      }
    }

    if (changed) this._onDidChange.fire();
  }
}
