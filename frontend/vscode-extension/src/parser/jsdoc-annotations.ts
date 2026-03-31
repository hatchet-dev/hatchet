import * as ts from 'typescript';

export interface WorkflowFactoryAnnotation {
  functionName: string;
  /**
   * Method on the returned object used to register tasks.
   * Sourced from `@hatchet-task-method <name>`. Defaults to `'task'`.
   */
  taskMethod: string;
  /**
   * Property name in the task options object that lists parent tasks.
   * Sourced from `@hatchet-task-parents <prop>`. Defaults to `'parents'`.
   */
  taskParentsProp: string;
}

/**
 * Scan a TypeScript source file for functions / arrow-function variables
 * annotated with `@hatchet-workflow` and return their metadata.
 *
 * Handles:
 *   - `/** @hatchet-workflow *\/ function foo(...) {}`
 *   - `/** @hatchet-workflow *\/ export function foo(...) {}`
 *   - `/** @hatchet-workflow *\/ const foo = (...) => {}`
 *   - `/** @hatchet-workflow *\/ const foo = function(...) {}`
 */
export function scanFileForWorkflowAnnotations(
  sourceText: string,
  fileName = 'file.ts',
): WorkflowFactoryAnnotation[] {
  // Quick bail — avoid full parse cost on files that have no annotation
  if (!sourceText.includes('@hatchet-workflow')) return [];

  const scriptKind = fileName.toLowerCase().endsWith('.tsx')
    ? ts.ScriptKind.TSX
    : ts.ScriptKind.TS;

  const sourceFile = ts.createSourceFile(
    fileName,
    sourceText,
    ts.ScriptTarget.ESNext,
    /*setParentNodes*/ true,
    scriptKind,
  );

  const results: WorkflowFactoryAnnotation[] = [];

  function visit(node: ts.Node): void {
    // function foo(...) {}  /  export function foo(...) {}
    if (ts.isFunctionDeclaration(node) && node.name) {
      const ann = tryExtract(node, node.name.text);
      if (ann) results.push(ann);
    }

    // const foo = (...) => {}  /  const foo = function(...) {}
    if (ts.isVariableStatement(node)) {
      const ann = tryExtractFromVariableStatement(node);
      if (ann) results.push(ann);
    }

    ts.forEachChild(node, visit);
  }

  visit(sourceFile);
  return results;
}

function tryExtract(node: ts.Node, functionName: string): WorkflowFactoryAnnotation | undefined {
  const tags = ts.getJSDocTags(node);
  if (!tags.some((t) => t.tagName.text === 'hatchet-workflow')) return undefined;

  return {
    functionName,
    taskMethod: getTagText(tags, 'hatchet-task-method') ?? 'task',
    taskParentsProp: getTagText(tags, 'hatchet-task-parents') ?? 'parents',
  };
}

function tryExtractFromVariableStatement(
  node: ts.VariableStatement,
): WorkflowFactoryAnnotation | undefined {
  // JSDoc attaches to the VariableStatement (before `const`/`let`)
  const tags = ts.getJSDocTags(node);
  if (!tags.some((t) => t.tagName.text === 'hatchet-workflow')) return undefined;

  for (const decl of node.declarationList.declarations) {
    if (!ts.isIdentifier(decl.name)) continue;
    const init = decl.initializer;
    if (!init) continue;
    if (ts.isArrowFunction(init) || ts.isFunctionExpression(init)) {
      return {
        functionName: decl.name.text,
        taskMethod: getTagText(tags, 'hatchet-task-method') ?? 'task',
        taskParentsProp: getTagText(tags, 'hatchet-task-parents') ?? 'parents',
      };
    }
  }
  return undefined;
}

function getTagText(tags: readonly ts.JSDocTag[], tagName: string): string | undefined {
  const tag = tags.find((t) => t.tagName.text === tagName);
  if (!tag) return undefined;
  const { comment } = tag;
  if (typeof comment === 'string') return comment.trim() || undefined;
  if (Array.isArray(comment)) {
    // Each element may be a JSDocText node (with a `.text` property) or a
    // JSDocLink node — use `.text` when present, otherwise skip.
    return (
      (comment as Array<{ text?: string }>)
        .map((c) => c.text ?? '')
        .join('')
        .trim() || undefined
    );
  }
  return undefined;
}
