import { hasFormFields, prepareSchema } from './prepare-schema';
import assert from 'node:assert/strict';
import { test } from 'node:test';

test('optional fields collapse to their non-null type', () => {
  const { schema } = prepareSchema({
    type: 'object',
    properties: {
      due: {
        anyOf: [{ type: 'string', format: 'date-time' }, { type: 'null' }],
        title: 'Due',
        default: null,
      },
    },
  });

  assert.deepEqual(schema.properties?.due, {
    type: 'string',
    format: 'date-time',
    title: 'Due',
  });
});

test('optional fields inside $defs collapse too', () => {
  const { schema } = prepareSchema({
    $defs: {
      Step: {
        type: 'object',
        properties: { n: { anyOf: [{ type: 'integer' }, { type: 'null' }] } },
      },
    },
    type: 'object',
    properties: { step: { $ref: '#/$defs/Step' } },
  });

  assert.deepEqual((schema.$defs?.Step as { properties: object }).properties, {
    n: { type: 'integer' },
  });
});

test('required nullable fields keep null as a named option', () => {
  const anyOf = [{ type: 'integer' }, { type: 'null' }];
  const { schema, uiSchema } = prepareSchema({
    type: 'object',
    properties: { count: { anyOf, title: 'Count' } },
    required: ['count'],
  });

  assert.deepEqual(schema.properties?.count, {
    anyOf: [{ type: 'integer' }, { type: 'null', default: null }],
    title: 'Count',
  });
  assert.deepEqual(uiSchema.count, {
    anyOf: [
      { 'ui:title': 'integer', 'ui:options': { label: false } },
      { 'ui:title': 'null' },
    ],
  });
});

test('unions that are not just nullable are left alone', () => {
  const anyOf = [{ type: 'integer' }, { type: 'string' }];
  const { schema } = prepareSchema({
    type: 'object',
    properties: { timeout: { anyOf } },
  });

  assert.deepEqual(schema.properties?.timeout, { anyOf });
});

test('the root title and description are dropped', () => {
  const { schema } = prepareSchema({
    title: 'Input',
    description: 'Long docstring',
    type: 'object',
    properties: {},
  });

  assert.equal(schema.title, undefined);
  assert.equal(schema.description, undefined);
});

test('arrays of primitives hide their per-item labels', () => {
  const { uiSchema } = prepareSchema({
    $defs: {
      Experiment: {
        type: 'object',
        properties: { tags: { type: 'array', items: { type: 'string' } } },
      },
    },
    type: 'object',
    properties: {
      labels: { type: 'array', items: { type: 'string' } },
      experiments: { type: 'array', items: { $ref: '#/$defs/Experiment' } },
    },
  });

  const stringItem = { 'ui:options': { label: false }, 'ui:emptyValue': '' };
  assert.deepEqual(uiSchema, {
    labels: { items: stringItem },
    experiments: { items: { tags: { items: stringItem } } },
  });
});

test('map values hide their label, including nested array item labels', () => {
  const { uiSchema } = prepareSchema({
    type: 'object',
    properties: {
      scopes: {
        type: 'object',
        additionalProperties: { type: 'array', items: { type: 'string' } },
      },
    },
  });

  assert.deepEqual(uiSchema, {
    scopes: {
      additionalProperties: {
        'ui:options': { label: false },
        items: { 'ui:options': { label: false }, 'ui:emptyValue': '' },
      },
    },
  });
});

test('string array items and map values default to an empty string', () => {
  const { schema } = prepareSchema({
    type: 'object',
    properties: {
      tags: { type: 'array', items: { type: 'string' } },
      labels: { type: 'object', additionalProperties: { type: 'string' } },
      counts: { type: 'array', items: { type: 'integer' } },
    },
  });

  const props = schema.properties as Record<string, Record<string, object>>;
  assert.deepEqual(props.tags.items, { type: 'string', default: '' });
  assert.deepEqual(props.labels.additionalProperties, {
    type: 'string',
    default: '',
  });
  assert.deepEqual(props.counts.items, { type: 'integer' });
});

test('enum array items and map values default to their first allowed value', () => {
  const enumItem = { type: 'string', enum: ['us-east-1', 'eu-west-1'] };
  const constItem = { type: 'string', const: 'eu-west-1' };
  const { schema } = prepareSchema({
    type: 'object',
    properties: {
      regions: { type: 'array', items: enumItem },
      pinned: { type: 'array', items: constItem },
      tiers: { type: 'object', additionalProperties: enumItem },
    },
  });

  const props = schema.properties as Record<string, Record<string, object>>;
  assert.deepEqual(props.regions.items, { ...enumItem, default: 'us-east-1' });
  assert.deepEqual(props.pinned.items, { ...constItem, default: 'eu-west-1' });
  assert.deepEqual(props.tiers.additionalProperties, {
    ...enumItem,
    default: 'us-east-1',
  });
});

test('literal values are not rewritten as schemas', () => {
  const config = {
    items: { type: 'string' },
    choice: { anyOf: [{ type: 'string' }, { type: 'null' }] },
  };
  const { schema } = prepareSchema({
    type: 'object',
    properties: {
      config: { type: 'object', default: config },
      mode: { const: config },
      kind: { enum: [config] },
    },
  });

  assert.deepEqual(schema.properties?.config, {
    type: 'object',
    default: config,
  });
  assert.deepEqual(schema.properties?.mode, { const: config });
  assert.deepEqual(schema.properties?.kind, { enum: [config] });
});

test('properties named like keywords are still walked as schemas', () => {
  const { schema } = prepareSchema({
    type: 'object',
    properties: {
      default: { anyOf: [{ type: 'string' }, { type: 'null' }] },
      enum: { type: 'array', items: { type: 'string' } },
    },
  });

  assert.deepEqual(schema.properties?.default, { type: 'string' });
  assert.deepEqual(schema.properties?.enum, {
    type: 'array',
    items: { type: 'string', default: '' },
  });
});

test('the input schema is not mutated', () => {
  const input = {
    title: 'Input',
    type: 'object',
    properties: { a: { anyOf: [{ type: 'string' }, { type: 'null' }] } },
  };
  const snapshot = JSON.stringify(input);

  prepareSchema(input);

  assert.equal(JSON.stringify(input), snapshot);
});

test('a model without fields has no form', () => {
  assert.equal(hasFormFields(undefined), false);
  assert.equal(hasFormFields({}), false);
  assert.equal(
    hasFormFields({ title: 'Empty', type: 'object', properties: {} }),
    false,
  );
  // dict[str, Any]: any key, any value, nothing for the form to describe
  assert.equal(
    hasFormFields({ type: 'object', additionalProperties: true }),
    false,
  );
});

test('fields, maps, non-object roots and $ref roots get a form', () => {
  assert.equal(
    hasFormFields({ type: 'object', properties: { n: { type: 'integer' } } }),
    true,
  );
  assert.equal(
    hasFormFields({
      type: 'object',
      additionalProperties: { type: 'string' },
    }),
    true,
  );
  assert.equal(
    hasFormFields({ type: 'array', items: { type: 'integer' } }),
    true,
  );
  assert.equal(
    hasFormFields({
      $ref: '#/$defs/Node',
      $defs: { Node: { type: 'object', properties: {} } },
    }),
    true,
  );
});
