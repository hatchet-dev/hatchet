import * as vscode from 'vscode';

/**
 * Return true if `uri` is under one of the currently open workspace folders.
 *
 * Uses `vscode.workspace.getWorkspaceFolder` rather than a raw string prefix
 * check to correctly handle directory boundary edges (e.g. a folder named
 * `/workspace/project` must not match `/workspace/project-evil/file.ts`).
 */
export function isWithinWorkspace(uri: vscode.Uri): boolean {
  return vscode.workspace.getWorkspaceFolder(uri) !== undefined;
}
