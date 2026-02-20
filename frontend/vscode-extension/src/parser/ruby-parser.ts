import type { ParsedTask, ParsedWorkflow, WorkflowDeclaration } from './workflow-parser';

/**
 * Collect the text inside the outermost `(...)` starting at the first `(` on
 * or after `lines[startLine]`.  Handles task calls that span multiple lines.
 */
function collectParenContent(lines: string[], startLine: number): string {
  const text = lines
    .slice(startLine, Math.min(startLine + 10, lines.length))
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
 * Parse a Ruby source file and return all Hatchet workflow declarations.
 *
 * Recognised patterns
 * -------------------
 * Workflow:  `CONST = <expr>.workflow(name: "Name")`
 * Task:      `[CONST = ]workflowVar.task(:name, ...) do`
 * Parents:   `parents: [STEP1, STEP2, :step3]`
 *
 * varId resolution
 * ----------------
 * - `STEP1 = workflow.task(:step1, ...)` → varId = `STEP1`, displayName = `step1`
 * - `workflow.task(:step3, ...)`         → varId = `step3`, displayName = `step3`
 * - Parent `STEP1`  → matches the task whose varId is `STEP1`
 * - Parent `:step3` → strip colon → matches the task whose varId is `step3`
 */
export function parseRubyWorkflows(source: string): ParsedWorkflow[] {
  const lines = source.split('\n');

  // ── Pass 1: workflow declarations ────────────────────────────────────────
  // e.g. `DAG_WORKFLOW = HATCHET.workflow(name: "DAGWorkflow")`
  const workflowVars = new Map<string, { name: string; declarationLine: number }>();
  const wfRe = /^(\w+)\s*=\s*\S+\.workflow\s*\(\s*name:\s*["']([^"']+)["']/;

  for (let i = 0; i < lines.length; i++) {
    const m = wfRe.exec(lines[i].trimStart());
    if (m) workflowVars.set(m[1], { name: m[2], declarationLine: i });
  }

  if (workflowVars.size === 0) return [];

  // ── Pass 2: task declarations ─────────────────────────────────────────────
  const tasksByWorkflow = new Map<string, ParsedTask[]>();

  // Match: [CONST = ]workflowVar.task(:taskName ...
  const taskRe = /^(?:(\w+)\s*=\s*)?(\w+)\.task\s*\(\s*:(\w+)/;

  for (let i = 0; i < lines.length; i++) {
    const taskM = taskRe.exec(lines[i].trimStart());
    if (!taskM) continue;

    const assignedConst = taskM[1]; // e.g. "STEP1" — may be undefined
    const workflowVar = taskM[2];
    const taskSymbolName = taskM[3]; // the bare name from the :symbol

    if (!workflowVars.has(workflowVar)) continue;

    // varId: use the assigned constant when present, otherwise the symbol name.
    // This ensures `:step3` parent references resolve correctly to the varId `step3`.
    const varId = assignedConst ?? taskSymbolName;
    const displayName = taskSymbolName;
    const declarationLine = i;

    // Collect full argument text to extract parents
    const args = collectParenContent(lines, i);

    // Extract parents: [STEP1, STEP2, :step3]
    const parentsM = /parents:\s*\[([^\]]*)\]/.exec(args);
    const parentVarIds: string[] = parentsM
      ? parentsM[1]
          .split(',')
          .map((s) => {
            const t = s.trim();
            return t.startsWith(':') ? t.slice(1) : t; // :step3 → step3; STEP1 → STEP1
          })
          .filter(Boolean)
      : [];

    if (!tasksByWorkflow.has(workflowVar)) tasksByWorkflow.set(workflowVar, []);
    tasksByWorkflow.get(workflowVar)!.push({
      varId,
      displayName,
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
export function detectRubyWorkflowDeclarations(source: string): WorkflowDeclaration[] {
  const lines = source.split('\n');
  const result: WorkflowDeclaration[] = [];
  // Regex is applied to trimStart() — varName starts at the indentation boundary
  const wfRe = /^(\w+)\s*=\s*\S+\.workflow\s*\(\s*name:\s*["']([^"']+)["']/;

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
