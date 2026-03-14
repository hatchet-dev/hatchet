import * as vscode from 'vscode';
import type { ParsedTask, ParsedWorkflow, WorkflowDeclaration } from '../parser/workflow-parser';

/**
 * Queries the running language server for all cross-file references to a
 * workflow variable and extracts task declarations from each reference site.
 *
 * Falls back to the single-file `fallbackTasks` when LSP returns nothing.
 */
export class LspAnalyzer {
  async analyzeWorkflow(
    decl: WorkflowDeclaration,
    documentUri: vscode.Uri,
    fallbackTasks: ParsedTask[],
    token: vscode.CancellationToken,
  ): Promise<{ workflow: ParsedWorkflow; usedFallback: boolean }> {
    const position = new vscode.Position(decl.declarationLine, decl.declarationCharacter);

    let locations: vscode.Location[] | undefined;
    try {
      locations = await vscode.commands.executeCommand<vscode.Location[]>(
        'vscode.executeReferenceProvider',
        documentUri,
        position,
      );
    } catch {
      // Language server not available — fall through to fallback
    }

    const makeFallback = (): { workflow: ParsedWorkflow; usedFallback: true } => ({
      workflow: {
        name: decl.name,
        varName: decl.varName,
        declarationLine: decl.declarationLine,
        tasks: fallbackTasks,
      },
      usedFallback: true,
    });

    if (!locations || locations.length === 0 || token.isCancellationRequested) {
      return makeFallback();
    }

    // Cap at 500 to avoid runaway LSP results (e.g. generated files, node_modules).
    const MAX_LOCATIONS = 500;
    const safeLocations = locations.slice(0, MAX_LOCATIONS).filter((loc) =>
      isWithinWorkspace(loc.uri),
    );

    if (safeLocations.length === 0) {
      return makeFallback();
    }

    const tasks: ParsedTask[] = [];

    for (const location of safeLocations) {
      if (token.isCancellationRequested) break;

      try {
        const doc = await vscode.workspace.openTextDocument(location.uri);
        const refLine = location.range.start.line;
        const startLine = Math.max(0, refLine - 2);
        const endLine = Math.min(doc.lineCount - 1, refLine + 30);

        const lines: string[] = [];
        for (let i = startLine; i <= endLine; i++) {
          lines.push(doc.lineAt(i).text);
        }

        const task = extractTaskAtLocation(
          lines,
          refLine - startLine, // offset of reference line within the slice
          startLine,           // absolute document line of lines[0]
          decl.varName,
          doc.languageId,
          location.uri,
        );

        if (task) {
          tasks.push(task);
        }
      } catch {
        // Skip locations that can't be opened
      }
    }

    if (tasks.length === 0) {
      return makeFallback();
    }

    // Validate parentVarIds — drop references to task varIds that weren't collected
    const taskVarIdSet = new Set(tasks.map((t) => t.varId));
    const validatedTasks = tasks.map((task) => ({
      ...task,
      parentVarIds: task.parentVarIds.filter((id) => taskVarIdSet.has(id)),
    }));

    return {
      workflow: {
        name: decl.name,
        varName: decl.varName,
        declarationLine: decl.declarationLine,
        tasks: validatedTasks,
      },
      usedFallback: false,
    };
  }
}

// ─── Workspace guard ─────────────────────────────────────────────────────────

/**
 * Return true if `uri` is under one of the currently open workspace folders.
 * Prevents acting on LSP-supplied URIs that point outside the workspace.
 */
function isWithinWorkspace(uri: vscode.Uri): boolean {
  const folders = vscode.workspace.workspaceFolders;
  if (!folders || folders.length === 0) return false;
  const uriStr = uri.toString();
  return folders.some((f) => uriStr.startsWith(f.uri.toString()));
}

// ─── Location-level task extraction ──────────────────────────────────────────

function extractTaskAtLocation(
  lines: string[],
  refLineOffset: number,
  absoluteStartLine: number,
  varName: string,
  languageId: string,
  fileUri: vscode.Uri,
): ParsedTask | undefined {
  switch (languageId) {
    case 'typescript':
    case 'typescriptreact':
    case 'javascript':
    case 'javascriptreact':
      return extractTsTask(lines, refLineOffset, absoluteStartLine, varName, fileUri);
    case 'python':
      return extractPyTask(lines, refLineOffset, absoluteStartLine, varName, fileUri);
    case 'go':
      return extractGoTask(lines, refLineOffset, absoluteStartLine, varName, fileUri);
    case 'ruby':
      return extractRubyTask(lines, refLineOffset, absoluteStartLine, varName, fileUri);
    default:
      return undefined;
  }
}

// ─── TypeScript ───────────────────────────────────────────────────────────────

function extractTsTask(
  lines: string[],
  refLineOffset: number,
  absoluteStartLine: number,
  varName: string,
  fileUri: vscode.Uri,
): ParsedTask | undefined {
  const refLine = lines[refLineOffset];

  // Match: [const varId = ]varName.task({
  const taskRe = new RegExp(
    `(?:const\\s+(\\w+)\\s*=\\s*)?${escapeRegex(varName)}\\.task\\s*\\(`,
  );
  const m = taskRe.exec(refLine);
  if (!m) return undefined;

  const taskVarId = m[1]; // may be undefined (anonymous task)

  // Build text from the reference line onward and locate the opening paren
  const fullText = lines.slice(refLineOffset).join('\n');
  const parenIdx = fullText.indexOf('(');
  if (parenIdx === -1) return undefined;

  const parenContent = collectBraceAwareContent(fullText, parenIdx);

  // Extract name: '...' or name: "..."
  const nameM = /name\s*:\s*['"`]([^'"`]+)['"`]/.exec(parenContent);
  const displayName = nameM?.[1] ?? taskVarId;
  if (!displayName) return undefined;

  const varId = taskVarId ?? sanitizeVarId(displayName);

  // Extract parents: [id1, id2]
  const parentsM = /parents\s*:\s*\[([^\]]*)\]/.exec(parenContent);
  const parentVarIds: string[] = parentsM
    ? parentsM[1]
        .split(',')
        .map((s) => s.trim())
        .filter((s) => /^\w+$/.test(s))
    : [];

  return {
    varId,
    displayName,
    parentVarIds,
    declarationLine: absoluteStartLine + refLineOffset,
    fileUri,
  };
}

// ─── Python ───────────────────────────────────────────────────────────────────

function extractPyTask(
  lines: string[],
  refLineOffset: number,
  absoluteStartLine: number,
  varName: string,
  fileUri: vscode.Uri,
): ParsedTask | undefined {
  const refLine = lines[refLineOffset];

  // Match: @varName.task(
  const decRe = new RegExp(`^@${escapeRegex(varName)}\\.task\\s*\\(`);
  if (!decRe.test(refLine.trimStart())) return undefined;

  // Collect decorator paren content
  const fullText = lines.slice(refLineOffset).join('\n');
  const parenIdx = fullText.indexOf('(');
  if (parenIdx === -1) return undefined;
  const parenContent = collectSimpleContent(fullText, parenIdx);

  // Extract parents=[id1, id2, ...]
  const parentsM = /parents\s*=\s*\[([^\]]*)\]/.exec(parenContent);
  const parentVarIds: string[] = parentsM
    ? parentsM[1]
        .split(',')
        .map((s) => s.trim())
        .filter((s) => /^\w+$/.test(s))
    : [];

  // Scan forward for `def` or `async def`
  let funcName: string | undefined;
  let defOffset = refLineOffset;
  for (let j = refLineOffset + 1; j < Math.min(refLineOffset + 10, lines.length); j++) {
    const defM = /^(?:async\s+)?def\s+(\w+)\s*\(/.exec(lines[j].trimStart());
    if (defM) {
      funcName = defM[1];
      defOffset = j;
      break;
    }
  }
  if (!funcName) return undefined;

  return {
    varId: funcName,
    displayName: funcName,
    parentVarIds,
    declarationLine: absoluteStartLine + defOffset,
    fileUri,
  };
}

// ─── Go ───────────────────────────────────────────────────────────────────────

function extractGoTask(
  lines: string[],
  refLineOffset: number,
  absoluteStartLine: number,
  varName: string,
  fileUri: vscode.Uri,
): ParsedTask | undefined {
  const refLine = lines[refLineOffset];

  // Match: [varId :=|_ =]varName.NewTask("name", ...
  const taskRe = new RegExp(
    `(?:(\\w+)\\s*(?::=|=)\\s*)?${escapeRegex(varName)}\\.NewTask\\s*\\(\\s*"([^"]+)"`,
  );
  const m = taskRe.exec(refLine.trimStart());
  if (!m) return undefined;

  const assignedVar = m[1];
  const taskName = m[2];
  const varId = assignedVar && assignedVar !== '_' ? assignedVar : sanitizeVarId(taskName);

  // Collect full NewTask(...) args with brace-aware depth
  const fullText = lines.slice(refLineOffset).join('\n');
  const parenIdx = fullText.indexOf('(');
  const taskArgs = parenIdx !== -1 ? collectBraceAwareContent(fullText, parenIdx) : '';

  // Extract WithParents(p1, p2)
  const parentsM = /WithParents\s*\(([^)]*)\)/.exec(taskArgs);
  const parentVarIds: string[] = parentsM
    ? parentsM[1]
        .split(',')
        .map((s) => s.trim())
        .filter((s) => /^\w+$/.test(s))
    : [];

  return {
    varId,
    displayName: taskName,
    parentVarIds,
    declarationLine: absoluteStartLine + refLineOffset,
    fileUri,
  };
}

// ─── Ruby ─────────────────────────────────────────────────────────────────────

function extractRubyTask(
  lines: string[],
  refLineOffset: number,
  absoluteStartLine: number,
  varName: string,
  fileUri: vscode.Uri,
): ParsedTask | undefined {
  const refLine = lines[refLineOffset];

  // Match: [CONST = ]varName.task(:name, ...
  const taskRe = new RegExp(
    `(?:(\\w+)\\s*=\\s*)?${escapeRegex(varName)}\\.task\\s*\\(\\s*:(\\w+)`,
  );
  const m = taskRe.exec(refLine.trimStart());
  if (!m) return undefined;

  const assignedConst = m[1];
  const taskSymbolName = m[2];
  const varId = assignedConst ?? taskSymbolName;

  // Collect paren content
  const fullText = lines.slice(refLineOffset).join('\n');
  const parenIdx = fullText.indexOf('(');
  if (parenIdx === -1) return undefined;
  const parenContent = collectSimpleContent(fullText, parenIdx);

  // Extract parents: [STEP1, STEP2, :step3]
  const parentsM = /parents:\s*\[([^\]]*)\]/.exec(parenContent);
  const parentVarIds: string[] = parentsM
    ? parentsM[1]
        .split(',')
        .map((s) => {
          const t = s.trim();
          return t.startsWith(':') ? t.slice(1) : t;
        })
        .filter(Boolean)
    : [];

  return {
    varId,
    displayName: taskSymbolName,
    parentVarIds,
    declarationLine: absoluteStartLine + refLineOffset,
    fileUri,
  };
}

// ─── Utilities ────────────────────────────────────────────────────────────────

/** Escape special regex characters in a literal string. */
function escapeRegex(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/** Sanitize a string to a valid JS identifier (used as varId fallback). */
function sanitizeVarId(name: string): string {
  return name.replace(/[^a-zA-Z0-9_$]/g, '_');
}

/**
 * Return the content inside the outermost `(...)` starting at `parenIdx`,
 * with brace-awareness so `{...}` blocks don't confuse the paren counter.
 * Suitable for TypeScript and Go.
 */
function collectBraceAwareContent(text: string, parenIdx: number): string {
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
 * Return the content inside the outermost `(...)` starting at `parenIdx`,
 * without brace-awareness. Suitable for Python and Ruby.
 */
function collectSimpleContent(text: string, parenIdx: number): string {
  let depth = 0;
  for (let i = parenIdx; i < text.length; i++) {
    if (text[i] === '(') depth++;
    else if (text[i] === ')') {
      if (--depth === 0) return text.slice(parenIdx + 1, i);
    }
  }
  return text.slice(parenIdx + 1);
}
