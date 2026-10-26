export type LabelRow = { key: string; value: string };

// whole numbers, without leading zeros, are saved as integer worker labels
const INTEGER_LABEL = /^-?(0|[1-9]\d*)$/;

export const labelsToRows = (
  labels: Record<string, unknown> | undefined,
): LabelRow[] =>
  Object.entries(labels ?? {}).map(([key, value]) => ({
    key,
    value: String(value),
  }));

/**
 * The labels to save from the form's rows. A value shown as the endpoint has it keeps its
 * type, so saving an edit leaves untouched labels as they were: the string "123" stays a
 * string. A new or changed value is saved as an integer when it is a whole number, and as
 * a string otherwise.
 */
export const rowsToLabels = (
  rows: LabelRow[],
  existing: Record<string, unknown> = {},
): Record<string, unknown> => {
  const labels: Record<string, unknown> = {};

  for (const { key, value } of rows) {
    const k = key.trim();

    if (k === '' && value.trim() === '') {
      continue;
    }

    const current = existing[k];

    if (current !== undefined && String(current) === value) {
      labels[k] = current;
      continue;
    }

    labels[k] = INTEGER_LABEL.test(value.trim()) ? Number(value.trim()) : value;
  }

  return labels;
};
