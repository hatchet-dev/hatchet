export function withNamespace(name: string, namespace?: string) {
  if (namespace && !name.startsWith(namespace)) {
    return `${namespace}${name}`;
  }

  return name;
}
