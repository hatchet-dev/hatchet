import type { ParsedTask, ParsedWorkflow, WorkflowDeclaration } from './workflow-parser';
import { collectWrapperNames, workflowNameFromCall } from './wrapper-annotations';

/**
 * Workflow declaration: `varName = <expr>.workflow(name=<X>)`.
 * `<X>` is either a quoted string literal (group 2) or any other expression
 * such as `stub.name` (group 3) — the latter is used verbatim as the label so
 * dynamically-named workflows still render.
 */
const WORKFLOW_RE =
  /^(\w+)\s*=\s*\S+\.workflow\s*\(\s*name\s*=\s*(?:["']([^"']+)["']|([^\s,)]+))/;

/** `varName = wrapperFn(...)` — wrapper-usage workflow declaration. */
const USAGE_RE = /^(\w+)\s*=\s*(\w+)\s*\(/;
/** `def funcName(` / `async def funcName(` — used to resolve marked wrappers. */
const DEF_RE = /^(?:async\s+)?def\s+(\w+)/;

/**
 * Collect the text inside the outermost `(...)` beginning at the first `(` on
 * or after `lines[startLine]`.  Handles decorators that span multiple lines.
 */
function collectParenContent(lines: string[], startLine: number): string {
  const text = lines
    .slice(startLine, Math.min(startLine + 20, lines.length))
    .join('\n');
  const parenStart = text.indexOf('(');
  if (parenStart === -1) return '';
  let depth = 0;
  let end = text.length - 1;
  for (let i = parenStart; i < text.length; i++) {
    if (text[i] === '(') depth++;
    else if (text[i] === ')') {
      if (--depth === 0) {
        end = i;
        break;
      }
    }
  }
  return text.slice(parenStart + 1, end);
}

/**
 * Parse a Python source file and return all Hatchet workflow declarations.
 *
 * Recognised patterns
 * -------------------
 * Workflow:  `varName = <expr>.workflow(name="WorkflowName")`
 *            `varName = <expr>.workflow(name=stub.name)`  (dynamic name → label
 *            falls back to the expression text; also matches indented
 *            declarations inside factory/wrapper functions)
 * Task:      `@varName.task(...)`  followed by  `[async] def funcName(...)`
 * Parents:   `parents=[step1, step2]`  (bare identifiers inside the list)
 */
export function parsePythonWorkflows(source: string): ParsedWorkflow[] {
  const lines = source.split('\n');

  // ── Pass 1: workflow declarations ────────────────────────────────────────
  const workflowVars = new Map<string, { name: string; declarationLine: number }>();
  // e.g. `dag_workflow = hatchet.workflow(name="DAGWorkflow")`
  const wrappers = collectWrapperNames(lines, DEF_RE);

  for (let i = 0; i < lines.length; i++) {
    const trimmed = lines[i].trimStart();
    const m = WORKFLOW_RE.exec(trimmed);
    if (m) {
      workflowVars.set(m[1], { name: m[2] ?? m[3], declarationLine: i });
      continue;
    }
    const u = USAGE_RE.exec(trimmed);
    if (u && wrappers.has(u[2])) {
      workflowVars.set(u[1], { name: workflowNameFromCall(trimmed, u[1]), declarationLine: i });
    }
  }

  if (workflowVars.size === 0) return [];

  // ── Pass 2: task decorators ───────────────────────────────────────────────
  const tasksByWorkflow = new Map<string, ParsedTask[]>();

  for (let i = 0; i < lines.length; i++) {
    // Match `@workflowVar.task(` — strip leading whitespace (handles indented code)
    const decM = /^@(\w+)\.task\s*\(/.exec(lines[i].trimStart());
    if (!decM || !workflowVars.has(decM[1])) continue;

    const workflowVar = decM[1];
    const decoratorLine = i;

    // Collect full decorator argument text (handles multi-line decorators)
    const args = collectParenContent(lines, i);

    // Extract parents=[id1, id2, ...] — only bare identifiers
    const parentsM = /parents\s*=\s*\[([^\]]*)\]/.exec(args);
    const parentVarIds: string[] = parentsM
      ? parentsM[1]
          .split(',')
          .map((s) => s.trim())
          .filter((s) => /^\w+$/.test(s))
      : [];

    // Scan ahead for `def` or `async def` within the next 10 lines
    let funcName: string | undefined;
    let defLine = decoratorLine;
    for (let j = i + 1; j < Math.min(i + 10, lines.length); j++) {
      const defM = /^(?:async\s+)?def\s+(\w+)\s*\(/.exec(lines[j].trimStart());
      if (defM) {
        funcName = defM[1];
        defLine = j;
        break;
      }
    }
    if (!funcName) continue;

    if (!tasksByWorkflow.has(workflowVar)) tasksByWorkflow.set(workflowVar, []);
    tasksByWorkflow.get(workflowVar)!.push({
      varId: funcName,
      displayName: funcName,
      parentVarIds,
      declarationLine: defLine, // jump-to-source lands on the `def` line
    });
  }

  // ── Build output ──────────────────────────────────────────────────────────
  return [...workflowVars.entries()].map(([varName, { name, declarationLine }]) => ({
    name,
    varName,
    declarationLine,
    tasks: tasksByWorkflow.get(varName) ?? [],
  }));
}

/**
 * Fast, Pass-1-only scan: return one `WorkflowDeclaration` per workflow
 * variable found in the source.  No task scanning — suitable for CodeLens.
 */
export function detectPyWorkflowDeclarations(source: string): WorkflowDeclaration[] {
  const lines = source.split('\n');
  const result: WorkflowDeclaration[] = [];
  const wrappers = collectWrapperNames(lines, DEF_RE);

  for (let i = 0; i < lines.length; i++) {
    const trimmed = lines[i].trimStart();
    const declarationCharacter = lines[i].length - trimmed.length;

    const m = WORKFLOW_RE.exec(trimmed);
    if (m) {
      result.push({ name: m[2] ?? m[3], varName: m[1], declarationLine: i, declarationCharacter });
      continue;
    }

    const u = USAGE_RE.exec(trimmed);
    if (u && wrappers.has(u[2])) {
      result.push({
        name: workflowNameFromCall(trimmed, u[1]),
        varName: u[1],
        declarationLine: i,
        declarationCharacter,
      });
    }
  }
  return result;
}
