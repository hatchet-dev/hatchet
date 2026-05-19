import { getErrorMessage } from './errors/hatchet-error';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function parseJSON<T = any>(json: string): T {
  try {
    const firstParse = JSON.parse(json);

    // Hatchet engine versions <=0.14.0 return JSON as a quoted string which needs to be parsed again.
    // This is a workaround for that issue, but will not be needed in future versions.
    try {
      return JSON.parse(firstParse) as T;
    } catch {
      return firstParse;
    }
  } catch (e: unknown) {
    throw new Error(`Could not parse JSON: ${getErrorMessage(e)}`, { cause: e });
  }
}
