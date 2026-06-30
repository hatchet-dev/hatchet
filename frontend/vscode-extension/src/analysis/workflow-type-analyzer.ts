import * as ts from 'typescript';
import * as path from 'path';

/**
 * Type-driven workflow detection.
 *
 * A "workflow" is anything whose type — after unwrapping `Promise<>` and
 * resolving aliases/inference — derives from one of the official Hatchet SDK
 * base declaration types. This follows arbitrary customer wrappers that a
 * syntactic scan cannot (e.g. `Promise<DurableWorkflow<I>>` where
 * `DurableWorkflow` is the customer's own alias for `WorkflowDeclaration`).
 *
 * Backed by a per-tsconfig `LanguageService` so repeated queries while editing
 * stay incremental.
 */

/** SDK base types that mean "this is a Hatchet workflow". */
const WORKFLOW_TYPE_NAMES = new Set([
  'BaseWorkflowDeclaration',
  'WorkflowDeclaration',
  'TaskWorkflowDeclaration',
]);

export interface WorkflowAnchor {
  /** Variable or function name the lens attaches to. */
  name: string;
  /** 0-based line of the name identifier. */
  declarationLine: number;
  /** 0-based column of the name identifier. */
  declarationCharacter: number;
  kind: 'variable' | 'function';
  /** 0-based last line of the enclosing function (functions only) — the body
   *  whose task calls form this anchor's DAG. */
  endLine?: number;
}

const DEFAULT_OPTIONS: ts.CompilerOptions = {
  target: ts.ScriptTarget.ESNext,
  module: ts.ModuleKind.CommonJS,
  moduleResolution: ts.ModuleResolutionKind.NodeJs,
  allowJs: false,
  noEmit: true,
  skipLibCheck: true,
};

interface ServiceEntry {
  service: ts.LanguageService;
  host: OverlayHost;
}

export class WorkflowTypeAnalyzer {
  /** One language service per tsconfig (keyed by its directory). */
  private readonly services = new Map<string, ServiceEntry>();

  /**
   * Return the workflow anchors in `filePath`, using `documentText` as the
   * (possibly unsaved) contents. Returns `null` when types cannot be resolved
   * (e.g. no tsconfig / TS unavailable) so callers can fall back to the
   * syntactic detector.
   */
  analyze(filePath: string, documentText: string): WorkflowAnchor[] | null {
    try {
      const entry = this.getService(filePath);
      if (!entry) return null;

      const normalized = normalize(filePath);
      entry.host.setOverlay(normalized, documentText);

      const program = entry.service.getProgram();
      if (!program) return null;
      const checker = program.getTypeChecker();
      const sourceFile = program.getSourceFile(normalized);
      if (!sourceFile) return null;

      return collectAnchors(sourceFile, checker);
    } catch {
      return null;
    }
  }

  dispose(): void {
    for (const { service } of this.services.values()) service.dispose();
    this.services.clear();
  }

  private getService(filePath: string): ServiceEntry | undefined {
    const configPath = ts.findConfigFile(path.dirname(filePath), ts.sys.fileExists, 'tsconfig.json');
    const key = configPath ? normalize(configPath) : `__none__:${path.dirname(filePath)}`;

    const existing = this.services.get(key);
    if (existing) return existing;

    let options = DEFAULT_OPTIONS;
    let rootNames: string[] = [];
    let baseDir = path.dirname(filePath);
    if (configPath) {
      baseDir = path.dirname(configPath);
      const parsed = ts.parseJsonConfigFileContent(
        ts.readConfigFile(configPath, ts.sys.readFile).config,
        ts.sys,
        baseDir,
      );
      options = { ...parsed.options, noEmit: true, skipLibCheck: true };
      rootNames = parsed.fileNames;
    }

    const host = new OverlayHost(rootNames, options, baseDir);
    const service = ts.createLanguageService(host, ts.createDocumentRegistry());
    const entry = { service, host };
    this.services.set(key, entry);
    return entry;
  }
}

/**
 * Language-service host that reads project files from disk but serves the
 * active document (and any other overlaid files) from memory.
 */
class OverlayHost implements ts.LanguageServiceHost {
  private readonly overlays = new Map<string, { version: number; text: string }>();
  private readonly roots: Set<string>;

  constructor(
    rootNames: string[],
    private readonly options: ts.CompilerOptions,
    private readonly baseDir: string,
  ) {
    this.roots = new Set(rootNames.map(normalize));
  }

  setOverlay(fileName: string, text: string): void {
    const key = normalize(fileName);
    this.roots.add(key);
    const prev = this.overlays.get(key);
    if (prev && prev.text === text) return;
    this.overlays.set(key, { version: (prev?.version ?? 0) + 1, text });
  }

  getScriptFileNames(): string[] {
    return [...this.roots];
  }

  getScriptVersion(fileName: string): string {
    const overlay = this.overlays.get(normalize(fileName));
    if (overlay) return `o${overlay.version}`;
    return ts.sys.getModifiedTime?.(fileName)?.getTime().toString() ?? '0';
  }

  getScriptSnapshot(fileName: string): ts.IScriptSnapshot | undefined {
    const overlay = this.overlays.get(normalize(fileName));
    if (overlay) return ts.ScriptSnapshot.fromString(overlay.text);
    if (!ts.sys.fileExists(fileName)) return undefined;
    const contents = ts.sys.readFile(fileName);
    return contents === undefined ? undefined : ts.ScriptSnapshot.fromString(contents);
  }

  getCurrentDirectory(): string {
    return this.baseDir;
  }

  getCompilationSettings(): ts.CompilerOptions {
    return this.options;
  }

  getDefaultLibFileName(options: ts.CompilerOptions): string {
    return ts.getDefaultLibFilePath(options);
  }

  fileExists = (f: string): boolean =>
    this.overlays.has(normalize(f)) || ts.sys.fileExists(f);
  readFile = (f: string): string | undefined =>
    this.overlays.get(normalize(f))?.text ?? ts.sys.readFile(f);
  readDirectory: ts.LanguageServiceHost['readDirectory'] = (...args) => ts.sys.readDirectory(...args);
  directoryExists = (d: string): boolean => ts.sys.directoryExists(d);
  getDirectories = (d: string): string[] => ts.sys.getDirectories(d);
}

function collectAnchors(sourceFile: ts.SourceFile, checker: ts.TypeChecker): WorkflowAnchor[] {
  const anchors: WorkflowAnchor[] = [];
  const seenNames = new Set<string>();

  const push = (nameNode: ts.Identifier, kind: WorkflowAnchor['kind'], bodyNode?: ts.Node): void => {
    if (seenNames.has(nameNode.text)) return;
    seenNames.add(nameNode.text);
    const pos = sourceFile.getLineAndCharacterOfPosition(nameNode.getStart(sourceFile));
    anchors.push({
      name: nameNode.text,
      declarationLine: pos.line,
      declarationCharacter: pos.character,
      kind,
      endLine: bodyNode
        ? sourceFile.getLineAndCharacterOfPosition(bodyNode.getEnd()).line
        : undefined,
    });
  };

  const returnsWorkflow = (decl: ts.SignatureDeclaration): boolean => {
    const sig = checker.getSignatureFromDeclaration(decl);
    if (!sig) return false;
    return isWorkflowType(checker.getReturnTypeOfSignature(sig), checker);
  };

  function visit(node: ts.Node): void {
    if (ts.isVariableStatement(node)) {
      for (const decl of node.declarationList.declarations) {
        if (!ts.isIdentifier(decl.name)) continue;
        // `const f = (): Workflow => ...` — the variable is a function; anchor
        // it when the function returns a workflow.
        const init = decl.initializer;
        if (init && (ts.isArrowFunction(init) || ts.isFunctionExpression(init))) {
          if (returnsWorkflow(init)) push(decl.name, 'function', init);
          continue;
        }
        if (isWorkflowType(checker.getTypeAtLocation(decl.name), checker)) {
          push(decl.name, 'variable');
        }
      }
    } else if (ts.isFunctionDeclaration(node) && node.name && returnsWorkflow(node)) {
      push(node.name, 'function', node);
    }
    ts.forEachChild(node, visit);
  }

  visit(sourceFile);

  // Dedupe: a function that returns a workflow already represents the DAG built
  // in its body, so drop variable anchors that live inside it (e.g. the inner
  // `const wf = ...`). Top-level variables are kept.
  const funcRanges = anchors
    .filter((a) => a.kind === 'function' && a.endLine !== undefined)
    .map((a) => [a.declarationLine, a.endLine!] as const);
  return anchors.filter(
    (a) =>
      a.kind === 'function' ||
      !funcRanges.some(([start, end]) => a.declarationLine >= start && a.declarationLine <= end),
  );
}

/** Unwrap a single `Promise<T>` layer, returning `T` (or undefined). */
function unwrapPromise(type: ts.Type, checker: ts.TypeChecker): ts.Type | undefined {
  if (type.getSymbol()?.getName() === 'Promise') {
    return checker.getTypeArguments(type as ts.TypeReference)[0];
  }
  return undefined;
}

/**
 * Whether `type` derives from a Hatchet workflow base type, following
 * `Promise<>` wrapping, alias resolution, and class inheritance.
 */
function isWorkflowType(type: ts.Type, checker: ts.TypeChecker, seen = new Set<ts.Type>()): boolean {
  if (!type || seen.has(type)) return false;
  seen.add(type);

  const promised = unwrapPromise(type, checker);
  if (promised && isWorkflowType(promised, checker, seen)) return true;

  const symbol = type.getSymbol() ?? type.aliasSymbol;
  if (symbol && WORKFLOW_TYPE_NAMES.has(symbol.getName()) && isFromHatchet(symbol)) {
    return true;
  }

  if (type.isClassOrInterface() || (type as ts.InterfaceType).getBaseTypes) {
    for (const base of type.getBaseTypes?.() ?? []) {
      if (isWorkflowType(base, checker, seen)) return true;
    }
  }

  if (type.isUnionOrIntersection()) {
    for (const t of type.types) if (isWorkflowType(t, checker, seen)) return true;
  }

  return false;
}

/** Guard against unrelated user types that happen to share a class name. */
function isFromHatchet(symbol: ts.Symbol): boolean {
  const decls = symbol.getDeclarations() ?? [];
  return decls.some((d) => {
    const file = d.getSourceFile().fileName;
    return file.includes('@hatchet-dev') || file.includes(`${path.sep}sdks${path.sep}typescript${path.sep}`);
  });
}

function normalize(fileName: string): string {
  return path.resolve(fileName).replace(/\\/g, '/');
}
