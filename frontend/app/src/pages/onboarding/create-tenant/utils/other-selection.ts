/**
 * Utility functions for handling "other" selections in form options
 */

interface OtherSelectionResult {
  isOtherSelected: boolean;
  otherValue: string;
  otherSelection: string | undefined;
}

/**
 * Extracts "other" selection information from an array of selected values
 * @param selectedValues Array of selected values that may contain "other: ..." entries
 * @returns Object containing other selection state and value
 */
export function extractOtherSelection(
  selectedValues: string[],
): OtherSelectionResult {
  const otherSelection = selectedValues.find((v) => v.startsWith('other'));
  const isOtherSelected = !!otherSelection;
  const otherValue = otherSelection
    ? otherSelection.replace('other: ', '')
    : '';

  return {
    isOtherSelected,
    otherValue,
    otherSelection,
  };
}

/**
 * Handles toggling the "other" option in a multi-select form
 * @param selectedValues Current array of selected values
 * @param isOtherSelected Whether "other" is currently selected
 * @param otherSelection The current "other" selection string
 * @returns Updated array of selected values
 */
export function toggleOtherOption(
  selectedValues: string[],
  isOtherSelected: boolean,
): string[] {
  if (isOtherSelected) {
    // Remove other option
    return selectedValues.filter((v) => !v.startsWith('other'));
  } else {
    // Add other option with empty value
    return [...selectedValues.filter((v) => !v.startsWith('other')), 'other: '];
  }
}

/**
 * Handles toggling a regular (non-other) option in a multi-select form
 * @param selectedValues Current array of selected values
 * @param optionValue The option value to toggle
 * @param isOtherSelected Whether "other" is currently selected
 * @param otherSelection The current "other" selection string
 * @returns Updated array of selected values
 */
export function toggleRegularOption(
  selectedValues: string[],
  optionValue: string,
  isOtherSelected: boolean,
  otherSelection?: string,
): string[] {
  if (selectedValues.includes(optionValue)) {
    // Remove the option
    return selectedValues.filter((v) => v !== optionValue);
  } else {
    // Add the option, preserving "other" if it exists
    return [
      ...selectedValues.filter((v) => !v.startsWith('other')),
      optionValue,
      ...(isOtherSelected && otherSelection ? [otherSelection] : []),
    ];
  }
}

/**
 * Updates the text value for an "other" selection
 * @param selectedValues Current array of selected values
 * @param text The new text value for the "other" option
 * @returns Updated array of selected values
 */
export function updateOtherText(
  selectedValues: string[],
  text: string,
): string[] {
  const newValues = selectedValues.filter((v) => !v.startsWith('other'));
  return [...newValues, `other: ${text}`];
}
