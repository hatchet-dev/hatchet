import { TokenCapability } from './api/generated/data-contracts';

export interface CapabilityCatalog {
  groups: { label: string; capabilities: TokenCapability[] }[];
  selectable: string[];
  // largest first, so a selection matching several bundles is sent as the broadest
  bundles: TokenCapability[];
  read: string[];
  write: string[];
}

export function buildCatalog(rows: TokenCapability[]): CapabilityCatalog {
  const groups: CapabilityCatalog['groups'] = [];

  for (const row of rows) {
    if (row.bundle) {
      continue;
    }

    const label = row.group ?? 'Other';
    const group = groups.find((g) => g.label === label);

    if (group) {
      group.capabilities.push(row);
    } else {
      groups.push({ label, capabilities: [row] });
    }
  }

  const selectable = groups.flatMap((g) => g.capabilities.map((c) => c.name));
  const bundles = rows
    .filter((r) => r.bundle)
    .sort((a, b) => b.includes.length - a.includes.length);

  // The read preset is the bundle that stands for less than everything; write is whatever it leaves out.
  const read =
    bundles.find((b) => b.includes.length < selectable.length)?.includes ?? [];

  return {
    groups,
    selectable,
    bundles,
    read,
    write: selectable.filter((c) => !read.includes(c)),
  };
}

// A selection matching a bundle goes out as that bundle, so the token keeps whatever the bundle grows to include.
export function toRequestedCapabilities(
  selected: string[],
  catalog: CapabilityCatalog,
): string[] {
  const chosen = new Set(selected);

  const bundle = catalog.bundles.find(
    (b) =>
      b.includes.length === chosen.size &&
      b.includes.every((c) => chosen.has(c)),
  );

  if (bundle) {
    return [bundle.name];
  }

  return catalog.selectable.filter((c) => chosen.has(c));
}

export function grantedCapabilities(
  selected: string[],
  catalog: CapabilityCatalog,
): Set<string> {
  const granted = new Set(selected);

  for (const group of catalog.groups) {
    for (const capability of group.capabilities) {
      if (granted.has(capability.name)) {
        capability.includes.forEach((c) => granted.add(c));
      }
    }
  }

  return granted;
}

export function describeTokenAccess(
  token: { capabilities?: string[]; readOnly: boolean },
  rows: TokenCapability[],
): string {
  // shards older than capabilities list tokens without them, and only know the readOnly flag
  if (!token.capabilities) {
    return token.readOnly ? 'Read (REST only)' : 'Read and write';
  }

  const bundle =
    token.capabilities.length === 1
      ? rows.find((r) => r.bundle && r.name === token.capabilities?.[0])
      : undefined;

  if (bundle?.label) {
    // the readOnly flag also keeps a token off gRPC, which the bundle alone does not
    return token.readOnly ? `${bundle.label} (REST only)` : bundle.label;
  }

  return token.capabilities.join(', ');
}
