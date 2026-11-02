/**
 * Recognizes a legacy v0 workflow and warns about it. This is a leaf module so that the
 * feature clients the core entry bundles can tell a legacy workflow apart without pulling in
 * the v0 transformer, which reaches the worker runtime.
 */
import { emitV0RemovedWarning } from '@util/v0-deprecation-warning';
import { BaseWorkflowDeclaration } from '../v1/declaration';
import type { Workflow } from './workflow';

const LEGACY_WORKFLOW_WARNING = [
  '',
  '\x1b[33m╔══════════════════════════════════════════════════════════════════════════════╗',
  '║                                                                              ║',
  '║   ⚠  DEPRECATION WARNING: Legacy workflow format detected.                   ║',
  '║                                                                              ║',
  '║   Please migrate to the v1 SDK:                                              ║',
  '║   https://docs.hatchet.run/home/migration-guide-typescript                   ║',
  '║                                                                              ║',
  '╚══════════════════════════════════════════════════════════════════════════════╝\x1b[0m',
  '',
].join('\n');

/**
 * Type guard: returns true if the workflow is a legacy v0 Workflow (not a BaseWorkflowDeclaration).
 */
export function isLegacyWorkflow(workflow: unknown): workflow is Workflow {
  return (
    workflow != null &&
    typeof workflow === 'object' &&
    !(workflow instanceof BaseWorkflowDeclaration) &&
    'id' in workflow &&
    'steps' in workflow &&
    Array.isArray((workflow as { steps?: unknown }).steps)
  );
}

let warnedLegacyWorkflow = false;

/**
 * Warns once per process that a v0 workflow is in use: the banner for the terminal and a
 * DeprecationWarning with a stable code that Node's warning flags and handlers can filter.
 */
export function warnLegacyWorkflow(): void {
  if (warnedLegacyWorkflow) {
    return;
  }

  warnedLegacyWorkflow = true;
  console.warn(LEGACY_WORKFLOW_WARNING);
  emitV0RemovedWarning('workflow');
}
