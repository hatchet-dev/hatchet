import * as ts from 'typescript';
import type * as vscode from 'vscode';
import type { WorkflowFactoryAnnotation } from './jsdoc-annotations';

export interface ParsedTask {
  varId: string;
  displayName: string;
  parentVarIds: string[];
  /** 0-based line of the task's `const x = wf.task({...})` or `wf.task({...})` statement */
  declarationLine: number;
  /** File where this task is declared — set by LspAnalyzer for cross-file tasks */
  fileUri?: vscode.Uri;
}

export interface ParsedWorkflow {
  /** The workflow's display name (from `.workflow({ name: '...' })`) */
  name: string;
  /** Variable name of the workflow object (e.g., `dag`) */
  varName: string;
  /** 0-based line number of the workflow declaration — used for CodeLens placement */
  declarationLine: number;
  tasks: ParsedTask[];
}

/**
 * Minimal workflow entry — enough for CodeLens. Tasks resolved later via LSP.
 */
export interface WorkflowDeclaration {
  /** Workflow display name */
  name: string;
  /** Source variable name (e.g. "dag", "DAG_WORKFLOW") */
  varName: string;
  /** 0-based line of the workflow declaration */
  declarationLine: number;
  /** 0-based column of the varName identifier — feeds LSP position query */
  declarationCharacter: number;
  /**
   * Present when this workflow variable was created by an `@hatchet-workflow`-
   * annotated factory function.  Carries the task-method and parents-prop names
   * needed for parsing tasks on the wrapper's returned object.
   */
  annotation?: WorkflowFactoryAnnotation;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

function getPropertyValue(
  obj: ts.ObjectLiteralExpression,
  key: string,
): ts.Expression | undefined {
  for (const prop of obj.properties) {
    if (
      ts.isPropertyAssignment(prop) &&
      ts.isIdentifier(prop.name) &&
      prop.name.text === key
    ) {
      return prop.initializer;
    }
  }
  return undefined;
}

function getStringLiteral(expr: ts.Expression | undefined): string | undefined {
  if (!expr) return undefined;
  if (ts.isStringLiteral(expr)) return expr.text;
  if (ts.isNoSubstitutionTemplateLiteral(expr)) return expr.text;
  return undefined;
}

/** Resolve the text of an identifier-like expression (simple Identifier only). */
function getIdentifierText(expr: ts.Expression): string | undefined {
  if (ts.isIdentifier(expr)) return expr.text;
  return undefined;
}

/**
 * Extract array element identifier names from an ArrayLiteralExpression.
 * E.g. `[toLower, reverse]` → `['toLower', 'reverse']`
 */
function extractParentIds(expr: ts.Expression): string[] {
  if (!ts.isArrayLiteralExpression(expr)) return [];
  return expr.elements
    .map((el) => getIdentifierText(el))
    .filter((id): id is string => id !== undefined);
}

/**
 * Return the identifier name of the outermost callee, ignoring any member-
 * access chain.  E.g. `foo(...)` → `'foo'`; `foo.bar(...)` → `'bar'`.
 * Returns `undefined` for more complex expressions.
 */
function getCalleeIdentifierName(call: ts.CallExpression): string | undefined {
  const callee = call.expression;
  if (ts.isIdentifier(callee)) return callee.text;
  if (ts.isPropertyAccessExpression(callee)) return callee.name.text;
  return undefined;
}

/**
 * Recursively search an object literal expression (and its nested object
 * literals) for a string-valued `name:` property.  Returns the first match.
 * Used to extract workflow names from arbitrarily-nested builder call args.
 */
function findNameInObjectLiteral(obj: ts.ObjectLiteralExpression): string | undefined {
  for (const prop of obj.properties) {
    if (!ts.isPropertyAssignment(prop)) continue;
    if (!ts.isIdentifier(prop.name)) continue;

    if (prop.name.text === 'name') {
      const val = getStringLiteral(prop.initializer);
      if (val) return val;
    }

    // Recurse into nested object literals
    if (ts.isObjectLiteralExpression(prop.initializer)) {
      const nested = findNameInObjectLiteral(prop.initializer);
      if (nested) return nested;
    }
  }
  return undefined;
}

/**
 * Search all object-literal arguments of a call expression, at any depth,
 * for a string-valued `name:` property.
 */
function extractWorkflowNameFromArgs(call: ts.CallExpression): string | undefined {
  for (const arg of call.arguments) {
    if (ts.isObjectLiteralExpression(arg)) {
      const found = findNameInObjectLiteral(arg);
      if (found) return found;
    }
  }
  return undefined;
}

// ─── Main parser ─────────────────────────────────────────────────────────────

/**
 * Parse a TypeScript source file and return all Hatchet workflow declarations.
 *
 * Two-pass approach:
 *   Pass 1 – collect workflow variable names:
 *     • `const X = *.workflow({ name })` (built-in Hatchet pattern)
 *     • `const X = annotatedFn(...)` where `annotatedFn` appears in
 *       `annotatedFunctions`
 *   Pass 2 – collect task declarations on known workflow variables, honouring
 *     each variable's annotation metadata for method name and parents prop.
 */
export function parseWorkflows(
  sourceText: string,
  fileName = 'workflow.ts',
  annotatedFunctions: ReadonlyMap<string, WorkflowFactoryAnnotation> = new Map(),
): ParsedWorkflow[] {
  const sourceFile = ts.createSourceFile(
    fileName,
    sourceText,
    ts.ScriptTarget.ESNext,
    /*setParentNodes*/ true,
    ts.ScriptKind.TS,
  );

  // ── Pass 1: find workflow declarations ──────────────────────────────────
  const workflowVars = new Map<
    string,
    { name: string; declarationLine: number; annotation?: WorkflowFactoryAnnotation }
  >();

  function visitForWorkflows(node: ts.Node): void {
    if (ts.isVariableStatement(node)) {
      for (const decl of node.declarationList.declarations) {
        if (!ts.isIdentifier(decl.name)) continue;
        const varName = decl.name.text;
        const init = decl.initializer;
        if (!init) continue;

        // ── Built-in: something.workflow({...}) or workflow({...})
        if (isWorkflowCall(init)) {
          const workflowName = extractWorkflowName(init);
          if (workflowName) {
            const line = sourceFile.getLineAndCharacterOfPosition(
              node.getStart(sourceFile),
            ).line;
            workflowVars.set(varName, { name: workflowName, declarationLine: line });
          }
        }

        // ── Annotated factory: createWorkflowBuilder({...})
        if (ts.isCallExpression(init)) {
          const calleeName = getCalleeIdentifierName(init);
          if (calleeName) {
            const ann = annotatedFunctions.get(calleeName);
            if (ann) {
              const workflowName =
                extractWorkflowNameFromArgs(init) ?? varName;
              const line = sourceFile.getLineAndCharacterOfPosition(
                node.getStart(sourceFile),
              ).line;
              workflowVars.set(varName, {
                name: workflowName,
                declarationLine: line,
                annotation: ann,
              });
            }
          }
        }
      }
    }

    ts.forEachChild(node, visitForWorkflows);
  }

  visitForWorkflows(sourceFile);

  if (workflowVars.size === 0) {
    return [];
  }

  // ── Pass 2: find task declarations ──────────────────────────────────────
  const tasksByWorkflow = new Map<string, ParsedTask[]>();
  const anonCounters = new Map<string, number>();

  function visitForTasks(node: ts.Node): void {
    let callExpr: ts.CallExpression | undefined;
    let taskVarId: string | undefined;
    let statementLine = 0;

    if (ts.isExpressionStatement(node) && ts.isCallExpression(node.expression)) {
      callExpr = node.expression;
      statementLine = sourceFile.getLineAndCharacterOfPosition(
        node.getStart(sourceFile),
      ).line;
    } else if (ts.isVariableStatement(node)) {
      for (const decl of node.declarationList.declarations) {
        if (
          ts.isIdentifier(decl.name) &&
          decl.initializer &&
          ts.isCallExpression(decl.initializer)
        ) {
          callExpr = decl.initializer;
          taskVarId = decl.name.text;
          statementLine = sourceFile.getLineAndCharacterOfPosition(
            node.getStart(sourceFile),
          ).line;
          break;
        }
      }
    }

    if (callExpr) {
      const parsed = tryParseTaskCall(callExpr, taskVarId, statementLine);
      if (parsed) {
        const { workflowVar, task } = parsed;
        if (!tasksByWorkflow.has(workflowVar)) {
          tasksByWorkflow.set(workflowVar, []);
        }
        tasksByWorkflow.get(workflowVar)!.push(task);
      }
    }

    ts.forEachChild(node, visitForTasks);
  }

  function tryParseTaskCall(
    call: ts.CallExpression,
    taskVarId: string | undefined,
    declarationLine: number,
  ): { workflowVar: string; task: ParsedTask } | undefined {
    const expr = call.expression;
    if (!ts.isPropertyAccessExpression(expr)) return undefined;

    const receiver = expr.expression;
    if (!ts.isIdentifier(receiver)) return undefined;
    const workflowVar = receiver.text;
    const wfEntry = workflowVars.get(workflowVar);
    if (!wfEntry) return undefined;

    const taskMethod = wfEntry.annotation?.taskMethod ?? 'task';
    const taskParentsProp = wfEntry.annotation?.taskParentsProp ?? 'parents';

    if (expr.name.text !== taskMethod) return undefined;

    const firstArg = call.arguments[0];
    if (!firstArg) return undefined;

    let displayName: string | undefined;
    let parentVarIds: string[] = [];

    if (ts.isStringLiteral(firstArg) || ts.isNoSubstitutionTemplateLiteral(firstArg)) {
      // Positional-name form: wf.task('step1', { parents: [...] })
      displayName = firstArg.text;
      const optsArg = call.arguments[1];
      if (optsArg && ts.isObjectLiteralExpression(optsArg)) {
        const parentsExpr = getPropertyValue(optsArg, taskParentsProp);
        parentVarIds = parentsExpr ? extractParentIds(parentsExpr) : [];
      }
    } else if (ts.isObjectLiteralExpression(firstArg)) {
      // Options-object form: wf.task({ name: 'step1', parents: [...] })
      const displayNameExpr = getPropertyValue(firstArg, 'name');
      displayName =
        getStringLiteral(displayNameExpr) ?? taskVarId ?? generateAnonId(workflowVar);
      const parentsExpr = getPropertyValue(firstArg, taskParentsProp);
      parentVarIds = parentsExpr ? extractParentIds(parentsExpr) : [];
    }

    if (!displayName) return undefined;

    const varId = taskVarId ?? sanitizeVarId(displayName);
    return { workflowVar, task: { varId, displayName, parentVarIds, declarationLine } };
  }

  function generateAnonId(workflowVar: string): string {
    const count = (anonCounters.get(workflowVar) ?? 0) + 1;
    anonCounters.set(workflowVar, count);
    return `${workflowVar}_task_${count}`;
  }

  visitForTasks(sourceFile);

  // ── Build output ─────────────────────────────────────────────────────────
  const results: ParsedWorkflow[] = [];

  for (const [varName, { name, declarationLine }] of workflowVars) {
    const tasks = tasksByWorkflow.get(varName) ?? [];
    results.push({ name, varName, declarationLine, tasks });
  }

  return results;
}

// ─── Shape helpers ───────────────────────────────────────────────────────────

/**
 * Determine whether a CallExpression matches the pattern `*.workflow({...})`.
 */
function isWorkflowCall(expr: ts.Expression): expr is ts.CallExpression {
  if (!ts.isCallExpression(expr)) return false;
  const callee = expr.expression;
  if (ts.isPropertyAccessExpression(callee) && callee.name.text === 'workflow') {
    return true;
  }
  if (ts.isIdentifier(callee) && callee.text === 'workflow') {
    return true;
  }
  return false;
}

/**
 * Extract the `name` string from a `.workflow({ name: '...' })` call.
 */
function extractWorkflowName(call: ts.CallExpression): string | undefined {
  const arg = call.arguments[0];
  if (!arg || !ts.isObjectLiteralExpression(arg)) return undefined;
  const nameProp = getPropertyValue(arg, 'name');
  return getStringLiteral(nameProp);
}

/** Sanitize a display name to a valid JS identifier (used as varId fallback). */
function sanitizeVarId(name: string): string {
  return name.replace(/[^a-zA-Z0-9_$]/g, '_');
}

// ─── Fast detection (Phase 1 / CodeLens) ─────────────────────────────────────

/**
 * Fast, Pass-1-only scan: return one `WorkflowDeclaration` per workflow
 * variable found in the source.  No task scanning — suitable for CodeLens.
 *
 * Detects both the built-in `.workflow({name})` pattern and calls to
 * `@hatchet-workflow`-annotated factory functions listed in
 * `annotatedFunctions`.
 */
export function detectTsWorkflowDeclarations(
  sourceText: string,
  fileName = 'workflow.ts',
  annotatedFunctions: ReadonlyMap<string, WorkflowFactoryAnnotation> = new Map(),
): WorkflowDeclaration[] {
  const sourceFile = ts.createSourceFile(
    fileName,
    sourceText,
    ts.ScriptTarget.ESNext,
    /*setParentNodes*/ true,
    ts.ScriptKind.TS,
  );

  const result: WorkflowDeclaration[] = [];

  function visit(node: ts.Node): void {
    if (ts.isVariableStatement(node)) {
      for (const decl of node.declarationList.declarations) {
        if (!ts.isIdentifier(decl.name)) continue;
        const varName = decl.name.text;
        const init = decl.initializer;
        if (!init) continue;

        // ── Built-in .workflow() detection ──────────────────────────────
        if (isWorkflowCall(init)) {
          const workflowName = extractWorkflowName(init);
          if (!workflowName) continue;

          const namePos = sourceFile.getLineAndCharacterOfPosition(
            decl.name.getStart(sourceFile),
          );
          result.push({
            name: workflowName,
            varName,
            declarationLine: namePos.line,
            declarationCharacter: namePos.character,
          });
          continue;
        }

        // ── Annotated factory detection ──────────────────────────────────
        if (ts.isCallExpression(init)) {
          const calleeName = getCalleeIdentifierName(init);
          if (!calleeName) continue;
          const ann = annotatedFunctions.get(calleeName);
          if (!ann) continue;

          const workflowName = extractWorkflowNameFromArgs(init) ?? varName;
          const namePos = sourceFile.getLineAndCharacterOfPosition(
            decl.name.getStart(sourceFile),
          );
          result.push({
            name: workflowName,
            varName,
            declarationLine: namePos.line,
            declarationCharacter: namePos.character,
            annotation: ann,
          });
        }
      }
    }
    ts.forEachChild(node, visit);
  }

  visit(sourceFile);
  return result;
}
