export function applyNamespace(name: string, namespace?: string) {
  if (namespace && !name.startsWith(namespace)) {
    return `${namespace}${name}`;
  }

  return name;
}

/**
 * Applies namespace to a workflow name and converts it to lowercase.
 * This ensures consistency with how workflow names are registered.
 */
export function normalizeWorkflowName(name: string, namespace?: string): string {
  return applyNamespace(name, namespace).toLowerCase();
}
