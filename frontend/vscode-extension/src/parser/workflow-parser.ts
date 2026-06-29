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
  name: string;
  varName: string;
  /** 0-based line number — used for CodeLens placement */
  declarationLine: number;
  tasks: ParsedTask[];
}

/** Minimal workflow entry — enough for CodeLens. Tasks resolved later via LSP. */
export interface WorkflowDeclaration {
  name: string;
  varName: string;
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

        // A direct `*.workflow(...)` call. Skip it when it lives inside an
        // annotated factory — its DAG is surfaced on the factory's usage site
        // instead, so we don't want a second lens on the inner call.
        if (isWorkflowCall(init) && !isInsideAnnotatedFactory(node, annotatedFunctions)) {
          const workflowName = resolveWorkflowName(init, sourceFile, varName);
          if (workflowName) {
            const line = sourceFile.getLineAndCharacterOfPosition(
              node.getStart(sourceFile),
            ).line;
            workflowVars.set(varName, { name: workflowName, declarationLine: line });
          }
        }

        // A usage of an annotated factory: `const x = factory(...)` or
        // `const x = await factory(...)`.
        const factoryCall = ts.isAwaitExpression(init) ? init.expression : init;
        if (ts.isCallExpression(factoryCall)) {
          const calleeName = getCalleeIdentifierName(factoryCall);
          if (calleeName) {
            const ann = annotatedFunctions.get(calleeName);
            if (ann) {
              const workflowName =
                extractWorkflowNameFromArgs(factoryCall) ?? varName;
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

  const results: ParsedWorkflow[] = [];

  for (const [varName, { name, declarationLine, annotation }] of workflowVars) {
    const collected = tasksByWorkflow.get(varName) ?? [];
    // When a factory usage has no tasks of its own, fall back to the DAG built
    // inside the factory body (captured on the annotation).
    const tasks = collected.length > 0 ? collected : annotation?.tasks ?? [];
    results.push({ name, varName, declarationLine, tasks });
  }

  return results;
}

/**
 * Walk up the AST from `node` to determine whether it lives inside a function
 * that is registered as an `@hatchet-workflow` factory. Used to suppress a
 * standalone lens on a `*.workflow(...)` call defined inside such a factory —
 * its DAG is surfaced on the factory's usage site instead.
 */
function isInsideAnnotatedFactory(
  node: ts.Node,
  annotatedFunctions: ReadonlyMap<string, WorkflowFactoryAnnotation>,
): boolean {
  if (annotatedFunctions.size === 0) return false;
  for (let cur = node.parent; cur; cur = cur.parent) {
    if (ts.isFunctionDeclaration(cur) && cur.name && annotatedFunctions.has(cur.name.text)) {
      return true;
    }
    if (
      (ts.isArrowFunction(cur) || ts.isFunctionExpression(cur)) &&
      cur.parent &&
      ts.isVariableDeclaration(cur.parent) &&
      ts.isIdentifier(cur.parent.name) &&
      annotatedFunctions.has(cur.parent.name.text)
    ) {
      return true;
    }
  }
  return false;
}

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
 * Resolve a workflow's display name from a `*.workflow({ name })` call.
 *
 * Returns `undefined` only when the first argument has no `name` property at
 * all — i.e. the call isn't a Hatchet workflow definition. When `name` is
 * present but not a static string (e.g. `name: stub.name` or a substituting
 * template), falls back to the name expression's source text so the workflow
 * still renders with a meaningful label, and finally to the variable name.
 */
function resolveWorkflowName(
  call: ts.CallExpression,
  sourceFile: ts.SourceFile,
  varNameFallback: string,
): string | undefined {
  const arg = call.arguments[0];
  if (!arg || !ts.isObjectLiteralExpression(arg)) return undefined;
  const nameExpr = getPropertyValue(arg, 'name');
  if (!nameExpr) return undefined;

  const literal = getStringLiteral(nameExpr);
  if (literal) return literal;

  return nameExpr.getText(sourceFile).trim() || varNameFallback;
}

/** Sanitize a display name to a valid JS identifier (used as varId fallback). */
function sanitizeVarId(name: string): string {
  return name.replace(/[^a-zA-Z0-9_$]/g, '_');
}

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

        // Direct `*.workflow(...)` — skip when inside an annotated factory
        // (surfaced on the factory's usage site instead).
        if (isWorkflowCall(init)) {
          if (isInsideAnnotatedFactory(node, annotatedFunctions)) continue;
          const workflowName = resolveWorkflowName(init, sourceFile, varName);
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

        // Usage of an annotated factory, optionally awaited.
        const factoryCall = ts.isAwaitExpression(init) ? init.expression : init;
        if (ts.isCallExpression(factoryCall)) {
          const calleeName = getCalleeIdentifierName(factoryCall);
          if (!calleeName) continue;
          const ann = annotatedFunctions.get(calleeName);
          if (!ann) continue;

          const workflowName = extractWorkflowNameFromArgs(factoryCall) ?? varName;
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
