import type { ParsedTask, ParsedWorkflow, WorkflowDeclaration } from './workflow-parser';

/**
 * Collect the argument text inside the outermost `(...)` of a `NewTask` call,
 * using brace-aware depth tracking so that function literals `func(...) {...}`
 * don't confuse the paren counter.
 */
function collectTaskCallArgs(lines: string[], startLine: number): string {
  const text = lines.slice(startLine, Math.min(startLine + 200, lines.length)).join('\n');
  const parenIdx = text.indexOf('(');
  if (parenIdx === -1) return '';

  let parenDepth = 0;
  let braceDepth = 0;

  for (let i = parenIdx; i < text.length; i++) {
    const ch = text[i];
    if (ch === '{') {
      braceDepth++;
    } else if (ch === '}') {
      if (braceDepth > 0) braceDepth--;
    } else if (braceDepth === 0) {
      if (ch === '(') parenDepth++;
      else if (ch === ')') {
        parenDepth--;
        if (parenDepth === 0) return text.slice(parenIdx + 1, i);
      }
    }
  }

  return text.slice(parenIdx + 1);
}

/**
 * Extract Go variable names from `hatchet.WithParents(p1, p2, ...)` within
 * the argument text of a single `NewTask` call.
 */
function extractWithParents(taskArgs: string): string[] {
  const m = /WithParents\s*\(([^)]*)\)/.exec(taskArgs);
  if (!m) return [];
  return m[1]
    .split(',')
    .map((s) => s.trim())
    .filter((s) => /^\w+$/.test(s));
}

/**
 * Parse a Go source file and return all Hatchet workflow declarations.
 *
 * Recognised patterns
 * -------------------
 * Workflow:  `varName := <expr>.NewWorkflow("WorkflowName")`
 * Task:      `[varName :=|_ =] workflowVar.NewTask("taskName", ..., hatchet.WithParents(p1, p2))`
 * Parents:   `hatchet.WithParents(step1, step2)` — bare Go variable identifiers
 *
 * varId resolution
 * ----------------
 * - `step1 := workflow.NewTask("step-1", ...)` → varId = `step1`, displayName = `step-1`
 * - `_ = workflow.NewTask("final-step", ...)`  → varId = `final-step`, displayName = `final-step`
 * - Parents are always Go variable identifiers matching other tasks' varIds.
 */
export function parseGoWorkflows(source: string): ParsedWorkflow[] {
  const lines = source.split('\n');

  // ── Pass 1: workflow declarations ────────────────────────────────────────
  // e.g. `workflow := client.NewWorkflow("dag-workflow")`
  const workflowVars = new Map<string, { name: string; declarationLine: number }>();
  const wfRe = /^(\w+)\s*:=\s*\S+\.NewWorkflow\s*\(\s*"([^"]+)"/;

  for (let i = 0; i < lines.length; i++) {
    const m = wfRe.exec(lines[i].trimStart());
    if (m) workflowVars.set(m[1], { name: m[2], declarationLine: i });
  }

  if (workflowVars.size === 0) return [];

  // ── Pass 2: task declarations ─────────────────────────────────────────────
  const tasksByWorkflow = new Map<string, ParsedTask[]>();

  // Match: [varName :=|_ =] workflowVar.NewTask("taskName", ...
  const taskRe = /^(?:(\w+)\s*(?::=|=)\s*)?(\w+)\.NewTask\s*\(\s*"([^"]+)"/;

  for (let i = 0; i < lines.length; i++) {
    const taskM = taskRe.exec(lines[i].trimStart());
    if (!taskM) continue;

    const assignedVar = taskM[1]; // e.g. "step1" or "_" — may be undefined
    const workflowVar = taskM[2];
    const taskName = taskM[3]; // e.g. "step-1"

    if (!workflowVars.has(workflowVar)) continue;

    // varId: use the assigned Go variable if present and not blank (_),
    // otherwise fall back to the task name string.
    const varId = assignedVar && assignedVar !== '_' ? assignedVar : taskName;
    const declarationLine = i;

    // Collect the full NewTask(...) argument text, then extract WithParents.
    // Using brace-aware collection ensures we don't bleed into the next task.
    const taskArgs = collectTaskCallArgs(lines, i);
    const parentVarIds = extractWithParents(taskArgs);

    if (!tasksByWorkflow.has(workflowVar)) tasksByWorkflow.set(workflowVar, []);
    tasksByWorkflow.get(workflowVar)!.push({
      varId,
      displayName: taskName,
      parentVarIds,
      declarationLine,
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
export function detectGoWorkflowDeclarations(source: string): WorkflowDeclaration[] {
  const lines = source.split('\n');
  const result: WorkflowDeclaration[] = [];
  // Regex is applied to trimStart() — varName starts at the indentation boundary
  const wfRe = /^(\w+)\s*:=\s*\S+\.NewWorkflow\s*\(\s*"([^"]+)"/;

  for (let i = 0; i < lines.length; i++) {
    const m = wfRe.exec(lines[i].trimStart());
    if (m) {
      const declarationCharacter = lines[i].length - lines[i].trimStart().length;
      result.push({
        name: m[2],
        varName: m[1],
        declarationLine: i,
        declarationCharacter,
      });
    }
  }
  return result;
}
