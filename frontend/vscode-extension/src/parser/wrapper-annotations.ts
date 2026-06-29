/**
 * Shared helpers for the comment-marker workflow-wrapper feature in the
 * regex-based parsers (Python, Go, Ruby).
 *
 * A *wrapper* is a factory function marked with `@hatchet-workflow` in a
 * comment on or just above its definition, e.g.
 *
 *   # @hatchet-workflow
 *   def make_workflow(hatchet, name):
 *       return hatchet.workflow(name=name)
 *
 * A *usage* of that wrapper (`wf = make_workflow(...)`) is then treated as a
 * workflow declaration, so tasks attached to the returned value (`wf.task(...)`)
 * render a DAG at the call site — mirroring the TypeScript `@hatchet-workflow`
 * JSDoc feature.
 */
export const WRAPPER_MARKER = '@hatchet-workflow';

/**
 * Collect the names of `@hatchet-workflow`-marked functions. `defRe` must match
 * a (trimmed) function-definition line and capture the function name in group 1.
 */
export function collectWrapperNames(lines: string[], defRe: RegExp): Set<string> {
  const names = new Set<string>();
  for (let i = 0; i < lines.length; i++) {
    if (!lines[i].includes(WRAPPER_MARKER)) continue;
    // The definition follows the marker, allowing for blank/decorator lines.
    for (let j = i; j < Math.min(i + 6, lines.length); j++) {
      const m = defRe.exec(lines[j].trimStart());
      if (m) {
        names.add(m[1]);
        break;
      }
    }
  }
  return names;
}

/**
 * Resolve a workflow label from a wrapper call's text: prefer a `name=`/`name:`
 * keyword literal, then the first positional string literal, else `fallback`
 * (the usage variable name).
 */
export function workflowNameFromCall(callText: string, fallback: string): string {
  const kw = /\bname\s*[:=]\s*["']([^"']+)["']/.exec(callText);
  if (kw) return kw[1];
  const positional = /["']([^"']+)["']/.exec(callText);
  return positional?.[1] ?? fallback;
}
