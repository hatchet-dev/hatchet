import * as vscode from 'vscode';
import {
  scanFileForWorkflowAnnotations,
  type WorkflowFactoryAnnotation,
} from '../parser/jsdoc-annotations';
import { parseWorkflows } from '../parser/workflow-parser';

// Paths containing these segments are ignored by the file-system watcher,
// matching the exclusion applied to the initial findFiles scan.
const IGNORED_PATH_SEGMENTS = ['node_modules', '.git'];

function isIgnoredUri(uri: vscode.Uri): boolean {
  const p = uri.fsPath;
  return IGNORED_PATH_SEGMENTS.some((seg) => p.includes(seg));
}

/**
 * For each `@hatchet-workflow` factory, capture the DAG built *inside* its body
 * so a usage of the factory can render it. Parsing with no annotations surfaces
 * the inner `*.workflow(...)` declaration (incl. dynamically-named ones); we
 * match it to a factory by line range.
 */
function enrichWithFactoryTasks(
  annotations: WorkflowFactoryAnnotation[],
  text: string,
  fileName: string,
): void {
  if (!annotations.some((a) => a.range)) return;
  const innerWorkflows = parseWorkflows(text, fileName);
  for (const ann of annotations) {
    const range = ann.range;
    if (!range) continue;
    const inner = innerWorkflows.find(
      (w) => w.declarationLine >= range.startLine && w.declarationLine <= range.endLine,
    );
    if (inner && inner.tasks.length > 0) ann.tasks = inner.tasks;
  }
}

/**
 * Workspace-level cache of `@hatchet-workflow`-annotated factory functions.
 *
 * Scans all `.ts` / `.tsx` workspace files on initialization, then stays
 * up-to-date via a file-system watcher.  Fires `onDidChange` whenever the
 * set of known annotations changes so callers can refresh (e.g. CodeLens).
 *
 * Internal storage is keyed by (fileKey → functionName) so that:
 *   • Removing a file only removes that file's annotations in O(k).
 *   • No `as any` casts are needed to track provenance.
 *   • Duplicate function names across files are handled deterministically
 *     (the flat view is rebuilt from scratch after every change).
 */
export class WorkflowAnnotationCache {
  private readonly _byFile = new Map<string, Map<string, WorkflowFactoryAnnotation>>();
  /** Flat view rebuilt after every mutation — returned by getAll(). */
  private _flat = new Map<string, WorkflowFactoryAnnotation>();

  private readonly _onDidChange = new vscode.EventEmitter<void>();
  readonly onDidChange: vscode.Event<void> = this._onDidChange.event;

  async initialize(context: vscode.ExtensionContext): Promise<void> {
    await this._scanWorkspace();

    const watcher = vscode.workspace.createFileSystemWatcher('**/*.{ts,tsx}');
    watcher.onDidChange((uri) => void this._scanUri(uri, /*remove*/ false));
    watcher.onDidCreate((uri) => void this._scanUri(uri, /*remove*/ false));
    watcher.onDidDelete((uri) => void this._scanUri(uri, /*remove*/ true));
    context.subscriptions.push(watcher, this._onDidChange);
  }

  getAll(): ReadonlyMap<string, WorkflowFactoryAnnotation> {
    return this._flat;
  }

  private async _scanWorkspace(): Promise<void> {
    const uris = await vscode.workspace.findFiles(
      '**/*.{ts,tsx}',
      '{**/node_modules/**,**/.git/**}',
    );
    await Promise.all(uris.map((uri) => this._scanUri(uri, false)));
  }

  private async _scanUri(uri: vscode.Uri, remove: boolean): Promise<void> {
    if (isIgnoredUri(uri)) return;

    const fileKey = uri.toString();
    const hadExisting = this._byFile.has(fileKey);

    // Remove stale entries for this file in O(k) — no full-map scan needed.
    this._byFile.delete(fileKey);

    let changed = hadExisting;

    if (!remove) {
      try {
        const bytes = await vscode.workspace.fs.readFile(uri);
        const text = new TextDecoder().decode(bytes);
        const found = scanFileForWorkflowAnnotations(text, uri.fsPath);
        if (found.length > 0) {
          enrichWithFactoryTasks(found, text, uri.fsPath);
          this._byFile.set(
            fileKey,
            new Map(found.map((ann) => [ann.functionName, ann])),
          );
          changed = true;
        }
      } catch {
        // File unreadable — silently skip
      }
    }

    if (changed) {
      this._rebuildFlat();
      this._onDidChange.fire();
    }
  }

  private _rebuildFlat(): void {
    const flat = new Map<string, WorkflowFactoryAnnotation>();
    for (const fileAnnotations of this._byFile.values()) {
      for (const [name, ann] of fileAnnotations) {
        flat.set(name, ann);
      }
    }
    this._flat = flat;
  }
}
