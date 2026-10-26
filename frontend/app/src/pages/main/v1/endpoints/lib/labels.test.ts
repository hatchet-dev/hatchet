import { labelsToRows, rowsToLabels } from './labels';
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

describe('rowsToLabels', () => {
  it('saves whole numbers as integers and the rest as strings', () => {
    assert.deepEqual(
      rowsToLabels([
        { key: 'region', value: 'us-east-1' },
        { key: 'weight', value: '10' },
        { key: 'padded', value: '007' },
        { key: '', value: '' },
      ]),
      { region: 'us-east-1', weight: 10, padded: '007' },
    );
  });

  it('keeps the type of a value the user left as it was', () => {
    const existing = { version: '123', weight: 10, canary: true };
    const rows = labelsToRows(existing);

    assert.deepEqual(rowsToLabels(rows, existing), existing);
  });

  it('retypes a value the user changed', () => {
    const existing = { version: '123' };

    assert.deepEqual(
      rowsToLabels([{ key: 'version', value: '124' }], existing),
      { version: 124 },
    );
  });
});
