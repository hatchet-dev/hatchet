import originalSnips, {
  Snippet as StdSnippet,
  snippets,
} from './generated/snips';

type Snippet = StdSnippet & {
  key: string;
};

export type { Snippet };
export { snippets };

// Helper type to get leaf values from snips object
type LeafValue = string;

// Helper type for the transformed object structure
type Snip<T> = {
  [P in keyof T]: T[P] extends LeafValue
    ? Snippet
    : T[P] extends object
      ? Snip<T[P]>
      : never;
};

// Function to transform the snips object
function transformSnips<T extends object>(obj: T): Snip<T> {
  const result: any = {};

  for (const [key, value] of Object.entries(obj)) {
    if (typeof value === 'string') {
      const [, path] = value.split(':');
      // For leaf nodes (strings), create an object with a get method
      result[key] = {
        key: value,
        ...snippets[path],
      };
    } else if (typeof value === 'object' && value !== null) {
      // Recursively transform nested objects
      result[key] = transformSnips(value);
    }
  }

  return result as Snip<T>;
}

// Transform the original snips object
const snips = transformSnips(originalSnips);

// Export the transformed object
export { snips };

export default snips;
