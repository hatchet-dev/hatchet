import * as ts from 'typescript';
import type * as vscode from 'vscode';
import { hasHatchetWorkflowTag, type WorkflowFactoryAnnotation } from './jsdoc-annotations';

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
  /**
   * When set, the DAG is built from this declaration's own scope — a function
   * body whose task calls are always in the same file. The panel resolves it
   * directly and skips the cross-file LSP reference step (which would find no
   * task references on a function name and mislabel it as a degraded fallback).
   */
  localOnly?: boolean;
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
 * Methods that register a task (a DAG node) on a Hatchet workflow/builder
 * object. Detection is *task-first*: any variable that receives one of these
 * calls is treated as a workflow, regardless of how it was constructed (direct
 * `hatchet.workflow(...)`, an awaited factory/builder, several wrappers deep).
 * Extend the set via an `@hatchet-task-method` annotation for custom methods.
 */
const DEFAULT_TASK_METHODS = ['task', 'durableTask'];

interface CollectedWorkflow {
  varName: string;
  name: string;
  declarationLine: number;
  declarationCharacter: number;
  tasks: ParsedTask[];
  annotation?: WorkflowFactoryAnnotation;
}

/**
 * Core, task-shape-based collection used by both the fast detector and the full
 * parser. A workflow is any variable that is *either* constructed as one
 * (`*.workflow(...)` / an annotated factory usage) *or* simply has task-method
 * calls attached to it. The DAG is built from those task calls; a constructed
 * workflow with no local tasks falls back to a factory's captured body tasks.
 */
function collectWorkflows(
  sourceFile: ts.SourceFile,
  annotatedFunctions: ReadonlyMap<string, WorkflowFactoryAnnotation>,
): CollectedWorkflow[] {
  const taskMethods = new Set<string>(DEFAULT_TASK_METHODS);
  for (const ann of annotatedFunctions.values()) {
    if (ann.taskMethod) taskMethods.add(ann.taskMethod);
  }

  const lineCharOf = (node: ts.Node) =>
    sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile));

  // ── Pass A: classify variable declarations ──────────────────────────────
  // Workflows discovered by construction, plus the position of every const/let
  // declaration (so a task-first variable can be placed on its declaration).
  const wfVars = new Map<
    string,
    { name: string; declarationLine: number; declarationCharacter: number; annotation?: WorkflowFactoryAnnotation }
  >();
  const declPos = new Map<string, { line: number; character: number }>();

  function visitDecls(node: ts.Node): void {
    if (ts.isVariableStatement(node)) {
      for (const decl of node.declarationList.declarations) {
        if (!ts.isIdentifier(decl.name)) continue;
        const varName = decl.name.text;
        const namePos = lineCharOf(decl.name);
        if (!declPos.has(varName)) {
          declPos.set(varName, { line: namePos.line, character: namePos.character });
        }
        const init = decl.initializer;
        if (!init) continue;

        // Direct `*.workflow(...)`. Skip when inside an annotated factory — its
        // DAG surfaces on the factory's usage site instead.
        if (isWorkflowCall(init) && !isInsideAnnotatedFactory(decl, annotatedFunctions)) {
          const name = resolveWorkflowName(init, sourceFile, varName);
          if (name) {
            wfVars.set(varName, {
              name,
              declarationLine: namePos.line,
              declarationCharacter: namePos.character,
            });
          }
          continue;
        }

        // Usage of an annotated factory: `const x = factory(...)` / `await factory(...)`.
        const call = ts.isAwaitExpression(init) ? init.expression : init;
        if (ts.isCallExpression(call)) {
          const calleeName = getCalleeIdentifierName(call);
          const ann = calleeName ? annotatedFunctions.get(calleeName) : undefined;
          if (ann) {
            wfVars.set(varName, {
              name: extractWorkflowNameFromArgs(call) ?? varName,
              declarationLine: namePos.line,
              declarationCharacter: namePos.character,
              annotation: ann,
            });
          }
        }
      }
    }
    ts.forEachChild(node, visitDecls);
  }
  visitDecls(sourceFile);

  // ── Pass B: collect task calls grouped by receiver variable ─────────────
  const tasksByVar = new Map<string, ParsedTask[]>();
  const firstTaskLine = new Map<string, number>();
  const anonCounters = new Map<string, number>();
  const genAnon = (v: string): string => {
    const c = (anonCounters.get(v) ?? 0) + 1;
    anonCounters.set(v, c);
    return `${v}_task_${c}`;
  };

  function recordTask(call: ts.CallExpression, taskVarId: string | undefined, line: number): void {
    const expr = call.expression;
    if (!ts.isPropertyAccessExpression(expr)) return;
    const receiver = expr.expression;
    if (!ts.isIdentifier(receiver)) return;
    if (!taskMethods.has(expr.name.text)) return;
    // Tasks defined inside an annotated factory belong to the factory's usage
    // site (surfaced via the captured annotation tasks), not the inner object.
    if (isInsideAnnotatedFactory(call, annotatedFunctions)) return;

    const receiverVar = receiver.text;
    const parentsProp = wfVars.get(receiverVar)?.annotation?.taskParentsProp ?? 'parents';

    let displayName: string | undefined;
    let parentVarIds: string[] = [];
    const firstArg = call.arguments[0];
    if (!firstArg) return;

    if (ts.isStringLiteral(firstArg) || ts.isNoSubstitutionTemplateLiteral(firstArg)) {
      // Positional-name form: wf.task('step1', { parents: [...] })
      displayName = firstArg.text;
      const optsArg = call.arguments[1];
      if (optsArg && ts.isObjectLiteralExpression(optsArg)) {
        const parentsExpr = getPropertyValue(optsArg, parentsProp);
        parentVarIds = parentsExpr ? extractParentIds(parentsExpr) : [];
      }
    } else if (ts.isObjectLiteralExpression(firstArg)) {
      // Options-object form: wf.task({ name: 'step1', parents: [...] })
      const nameExpr = getPropertyValue(firstArg, 'name');
      displayName = getStringLiteral(nameExpr) ?? taskVarId ?? genAnon(receiverVar);
      const parentsExpr = getPropertyValue(firstArg, parentsProp);
      parentVarIds = parentsExpr ? extractParentIds(parentsExpr) : [];
    } else {
      return;
    }
    if (!displayName) return;

    const varId = taskVarId ?? sanitizeVarId(displayName);
    if (!tasksByVar.has(receiverVar)) tasksByVar.set(receiverVar, []);
    tasksByVar.get(receiverVar)!.push({ varId, displayName, parentVarIds, declarationLine: line });
    if (!firstTaskLine.has(receiverVar)) firstTaskLine.set(receiverVar, line);
  }

  function visitTasks(node: ts.Node): void {
    if (ts.isExpressionStatement(node) && ts.isCallExpression(node.expression)) {
      recordTask(node.expression, undefined, lineCharOf(node).line);
    } else if (ts.isVariableStatement(node)) {
      for (const decl of node.declarationList.declarations) {
        if (ts.isIdentifier(decl.name) && decl.initializer && ts.isCallExpression(decl.initializer)) {
          recordTask(decl.initializer, decl.name.text, lineCharOf(node).line);
          break;
        }
      }
    }
    ts.forEachChild(node, visitTasks);
  }
  visitTasks(sourceFile);

  // ── Merge: a workflow is either constructed as one or has tasks attached ──
  const names = new Set<string>([...wfVars.keys(), ...tasksByVar.keys()]);
  const results: CollectedWorkflow[] = [];
  for (const varName of names) {
    const wf = wfVars.get(varName);
    const collected = tasksByVar.get(varName) ?? [];
    const tasks = collected.length > 0 ? collected : wf?.annotation?.tasks ?? [];

    let name: string;
    let declarationLine: number;
    let declarationCharacter: number;
    if (wf) {
      ({ name, declarationLine, declarationCharacter } = wf);
    } else {
      // Task-first only: a variable that merely receives task calls.
      const pos = declPos.get(varName);
      name = varName;
      declarationLine = pos?.line ?? firstTaskLine.get(varName) ?? 0;
      declarationCharacter = pos?.character ?? 0;
    }
    results.push({ varName, name, declarationLine, declarationCharacter, tasks, annotation: wf?.annotation });
  }
  return results;
}

/**
 * Parse a TypeScript source file and return all Hatchet workflow declarations
 * together with their task DAGs.
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

  return collectWorkflows(sourceFile, annotatedFunctions).map((w) => ({
    name: w.name,
    varName: w.varName,
    declarationLine: w.declarationLine,
    tasks: w.tasks,
  }));
}

/**
 * Walk up the AST from `node` to determine whether it lives inside a function
 * carrying a local `@hatchet-workflow` JSDoc tag. Used to suppress a standalone
 * lens on a `*.workflow(...)` call defined inside such a factory — its DAG is
 * surfaced on the factory's usage site instead.
 *
 * The annotation is read from the enclosing node's own JSDoc (not the
 * workspace-wide name map) so an unannotated function that merely shares a name
 * with an annotated one elsewhere is never affected. `annotatedFunctions` is
 * only a fast bail: when empty there is nothing to suppress — and the annotation
 * cache relies on this when it parses a factory body (with no annotations) to
 * capture the very inner workflow this would otherwise hide.
 */
function isInsideAnnotatedFactory(
  node: ts.Node,
  annotatedFunctions: ReadonlyMap<string, WorkflowFactoryAnnotation>,
): boolean {
  if (annotatedFunctions.size === 0) return false;
  for (let cur = node.parent; cur; cur = cur.parent) {
    // function foo(...) {}
    if (ts.isFunctionDeclaration(cur) && hasHatchetWorkflowTag(cur)) {
      return true;
    }
    // const foo = (...) => {} / const foo = function(...) {} — JSDoc sits on the
    // enclosing variable statement.
    if (ts.isArrowFunction(cur) || ts.isFunctionExpression(cur)) {
      const stmt = cur.parent?.parent?.parent;
      if (stmt && ts.isVariableStatement(stmt) && hasHatchetWorkflowTag(stmt)) {
        return true;
      }
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
 * Fast scan: return one `WorkflowDeclaration` per workflow variable found —
 * suitable for CodeLens placement. Task-shape-based, so it also surfaces
 * workflows reached through awaited factories/builders, not just direct
 * `.workflow({name})` calls or annotated factories.
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

  return collectWorkflows(sourceFile, annotatedFunctions).map((w) => ({
    name: w.name,
    varName: w.varName,
    declarationLine: w.declarationLine,
    declarationCharacter: w.declarationCharacter,
    annotation: w.annotation,
  }));
}
