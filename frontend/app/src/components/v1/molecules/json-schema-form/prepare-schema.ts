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

// Values under these keys are input data rather than schemas, so the walks
// below must not rewrite them: a default of `{"items": {"type": "string"}}`
// is a value for the workflow, not a list.
const LITERAL_KEYS = new Set(['default', 'const', 'enum', 'examples']);

// Values under these keys map names to schemas, so the names themselves are
// not schema keywords (a property may well be called `default`).
const SCHEMA_MAP_KEYS = new Set([
  'properties',
  'patternProperties',
  '$defs',
  'definitions',
]);

// Returns a copy of `node` with `fn` applied to each direct subschema.
// `property` is set for the members of `properties`.
function mapSubschemas(
  node: SchemaNode,
  fn: (schema: unknown, property?: string) => unknown,
): SchemaNode {
  return Object.fromEntries(
    Object.entries(node).map(([key, value]) => {
      if (LITERAL_KEYS.has(key)) {
        return [key, value];
      }
      if (SCHEMA_MAP_KEYS.has(key) && isNode(value)) {
        return [
          key,
          Object.fromEntries(
            Object.entries(value).map(([name, child]) => [
              name,
              fn(child, key === 'properties' ? name : undefined),
            ]),
          ),
        ];
      }
      return [key, Array.isArray(value) ? value.map((v) => fn(v)) : fn(value)];
    }),
  );
}

// the exact `X | None` shape; real unions such as `int | str` (or
// `int | str | None`) are not nullable in this sense
function isNullable(node: SchemaNode): node is SchemaNode & {
  anyOf: [unknown, unknown];
} {
  return (
    Array.isArray(node.anyOf) &&
    node.anyOf.length === 2 &&
    node.anyOf.filter(isNullType).length === 1
  );
}

// pydantic emits `X | None` as `anyOf: [X, {type: null}]`, which rjsf renders
// as a dropdown of "option 1 / option 2"; render it as a plain optional X.
// A required `X | None` (no default in pydantic) keeps the union, since null
// is a value the workflow must be able to receive and an empty input would
// only ever send nothing; it starts on null, and buildUiSchema names its
// options.
function collapseNullable(node: unknown, required = false): unknown {
  if (!isNode(node)) {
    return node;
  }

  const requiredProperties = Array.isArray(node.required) ? node.required : [];
  const walked = mapSubschemas(node, (child, property) =>
    collapseNullable(
      child,
      property !== undefined && requiredProperties.includes(property),
    ),
  );

  if (!isNullable(walked)) {
    return walked;
  }

  if (required) {
    // without a default, rjsf leaves the value undefined when the null option
    // is picked, and the field is then left out of the input entirely
    return {
      ...walked,
      anyOf: walked.anyOf.map((option) =>
        isNullType(option)
          ? { ...(option as SchemaNode), default: null }
          : option,
      ),
    };
  }

  const { anyOf, ...rest } = walked;
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
function seedStringDefaults(node: unknown): unknown {
  if (!isNode(node)) {
    return node;
  }
  const walked = mapSubschemas(node, seedStringDefaults);
  for (const key of ['items', 'additionalProperties']) {
    const child = walked[key];
    if (!isNode(child) || child.type !== 'string' || 'default' in child) {
      continue;
    }
    // '' is not an allowed value of an enum (e.g. a list of Literals), so
    // those start on their first allowed value instead
    let seeded: unknown = '';
    if ('const' in child) {
      seeded = child.const;
    } else if (Array.isArray(child.enum) && child.enum.length > 0) {
      seeded = child.enum[0];
    }
    walked[key] = { ...child, default: seeded };
  }
  return walked;
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

  // a required `X | None` kept by collapseNullable: name its options by type
  // rather than "Count option 1 / option 2", and drop the inner field's label,
  // which would repeat the one above the option picker
  if (isNullable(resolved)) {
    ui.anyOf = resolved.anyOf.map((option) => {
      const target = isNode(option) ? resolveRef(option, root) : {};
      if (isNullType(target)) {
        return { 'ui:title': 'null' };
      }
      const optionUi = buildUiSchema(target, root, depth + 1);
      return {
        ...optionUi,
        'ui:title': String(target.title ?? target.type ?? 'value'),
        'ui:options': { ...optionUi['ui:options'], label: false },
      };
    });
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

// Whether the schema has anything to fill in. SDK schemas always carry
// metadata (title, type), so a model without fields still arrives as
// `{"type": "object", "properties": {}}`; that gets no form. Maps, non-object
// roots and `$ref` roots (pydantic's shape for recursive models) have no
// top-level `properties` but do render.
export function hasFormFields(schema: unknown): schema is SchemaNode {
  if (!isNode(schema)) {
    return false;
  }
  if (['$ref', 'anyOf', 'oneOf', 'allOf'].some((key) => key in schema)) {
    return true;
  }
  if (schema.type !== undefined && schema.type !== 'object') {
    return true;
  }
  return (
    (isNode(schema.properties) && Object.keys(schema.properties).length > 0) ||
    isNode(schema.additionalProperties)
  );
}

export function prepareSchema(schema: object): {
  schema: RJSFSchema;
  uiSchema: UiSchema;
} {
  // both passes return copies, so the deletes below leave `schema` alone
  const prepared = seedStringDefaults(collapseNullable(schema)) as SchemaNode;

  // the dialog already names the workflow, and a model docstring is rarely
  // written for the person filling the form
  delete prepared.title;
  delete prepared.description;

  return {
    schema: prepared as RJSFSchema,
    uiSchema: buildUiSchema(prepared, prepared),
  };
}
