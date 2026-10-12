// Adapts a workflow's inputJsonSchema before it is handed to rjsf.
//
// The schemas come from the SDKs (pydantic's `json_schema()` in Python, zod's
// `toJSONSchema()` in TypeScript) and are valid JSON Schema, but rjsf renders
// some of their idioms poorly: nullable fields become an "option 1 / option 2"
// dropdown, empty list items fail validation before anything is typed, and
// every array item or map value repeats its parent's label. This module
// rewrites those patterns and builds the matching uiSchema, so the form stays
// usable without each SDK having to emit rjsf-specific schemas.
import type { RJSFSchema, UiSchema } from '@rjsf/utils';

type SchemaNode = Record<string, unknown>;

const isNode = (value: unknown): value is SchemaNode =>
  typeof value === 'object' && value !== null && !Array.isArray(value);

const isNullType = (value: unknown) => isNode(value) && value.type === 'null';

const PRIMITIVE_TYPES = new Set(['string', 'number', 'integer', 'boolean']);

// pydantic emits `X | None` as `anyOf: [X, {type: null}]`, which rjsf renders
// as a dropdown of "option 1 / option 2"; render it as a plain optional X
function collapseNullable(node: unknown): unknown {
  if (Array.isArray(node)) {
    return node.map(collapseNullable);
  }
  if (!isNode(node)) {
    return node;
  }

  const walked: SchemaNode = Object.fromEntries(
    Object.entries(node).map(([key, value]) => [key, collapseNullable(value)]),
  );

  const { anyOf, ...rest } = walked;
  // only the exact `X | None` shape is collapsed; real unions such as
  // `int | str` (or `int | str | None`) keep rjsf's option picker
  if (
    !Array.isArray(anyOf) ||
    anyOf.length !== 2 ||
    anyOf.filter(isNullType).length !== 1
  ) {
    return walked;
  }

  // `rest` holds the field-level keys (title, description, default) that sit
  // next to the anyOf, and they win over the inner type's own
  const merged: SchemaNode = {
    ...(anyOf.find((option) => !isNullType(option)) as SchemaNode),
    ...rest,
  };
  // pydantic gives optional fields `default: null`, which no longer matches the
  // collapsed type and would show a validation error on an untouched form
  if (merged.default === null) {
    delete merged.default;
  }
  return merged;
}

// rjsf resolves $refs itself when rendering; this is only so the uiSchema walk
// can see what a referenced model contains. pydantic and zod both put shared
// definitions under the root `$defs`, so other ref forms are left alone. Keys
// written next to the $ref (title, description, default) override the
// definition's, as they do when rjsf resolves the ref.
function resolveRef(node: SchemaNode, root: SchemaNode): SchemaNode {
  const ref = node.$ref;
  if (typeof ref !== 'string' || !ref.startsWith('#/$defs/')) {
    return node;
  }
  const defs = isNode(root.$defs) ? root.$defs : {};
  const target = defs[ref.slice('#/$defs/'.length)];
  return isNode(target) ? { ...target, ...node } : node;
}

// rjsf leaves a new array item or map value undefined until typed into, which
// fails validation ("must be string") before the user has done anything.
// Mutates in place: only call it on the copy made by collapseNullable.
function seedStringDefaults(node: unknown): void {
  if (Array.isArray(node)) {
    node.forEach(seedStringDefaults);
    return;
  }
  if (!isNode(node)) {
    return;
  }
  for (const key of ['items', 'additionalProperties']) {
    const child = node[key];
    if (isNode(child) && child.type === 'string' && !('default' in child)) {
      child.default = '';
    }
  }
  Object.values(node).forEach(seedStringDefaults);
}

function withoutLabel(ui: UiSchema, node: SchemaNode): UiSchema {
  return {
    ...ui,
    'ui:options': { ...ui['ui:options'], label: false },
    // clearing a string input would otherwise set undefined and fail validation
    ...(node.type === 'string' ? { 'ui:emptyValue': '' } : {}),
  };
}

// keeps the uiSchema free of empty `{}` entries for fields that need nothing
const hasKeys = (ui: UiSchema) => Object.keys(ui).length > 0;

// Labels that only repeat their container are hidden: array items of
// primitives ("Tags-1" under "Tags") and map values (the key input already
// names them). Object items in arrays keep their title as a section heading.
//
// A uiSchema mirrors the shape of the data (keyed by property name, with
// `items` for array elements and `additionalProperties` for map values), so the
// walk follows those three keys.
function buildUiSchema(
  node: SchemaNode,
  root: SchemaNode,
  depth = 0,
): UiSchema {
  // a recursive model (one that references itself through $defs) would
  // otherwise be walked forever; nothing that deep needs custom ui options
  if (depth > 10) {
    return {};
  }

  const resolved = resolveRef(node, root);
  const ui: UiSchema = {};

  if (isNode(resolved.properties)) {
    for (const [key, child] of Object.entries(resolved.properties)) {
      if (!isNode(child)) {
        continue;
      }
      const childUi = buildUiSchema(child, root, depth + 1);
      if (hasKeys(childUi)) {
        ui[key] = childUi;
      }
    }
  }

  if (isNode(resolved.additionalProperties)) {
    const value = resolveRef(resolved.additionalProperties, root);
    ui.additionalProperties = withoutLabel(
      buildUiSchema(value, root, depth + 1),
      value,
    );
  }

  if (isNode(resolved.items)) {
    const items = resolveRef(resolved.items, root);
    const itemsUi = buildUiSchema(items, root, depth + 1);
    if (PRIMITIVE_TYPES.has(items.type as string)) {
      ui.items = withoutLabel(itemsUi, items);
    } else if (hasKeys(itemsUi)) {
      ui.items = itemsUi;
    }
  }

  return ui;
}

export function prepareSchema(schema: object): {
  schema: RJSFSchema;
  uiSchema: UiSchema;
} {
  // collapseNullable returns a copy, so the steps below can mutate it
  const prepared = collapseNullable(schema) as SchemaNode;
  seedStringDefaults(prepared);

  // the dialog already names the workflow, and a model docstring is rarely
  // written for the person filling the form
  delete prepared.title;
  delete prepared.description;

  return {
    schema: prepared as RJSFSchema,
    uiSchema: buildUiSchema(prepared, prepared),
  };
}
