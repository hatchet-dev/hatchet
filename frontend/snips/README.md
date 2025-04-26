# Snips Library

A TypeScript utility library for use within the Hatchet monorepo.

## Installation

This package is intended for internal use within the Hatchet monorepo. You can add it to your project by adding it as a dependency in your package.json:

```json
{
  "dependencies": {
    "@hatchet/snips": "workspace:*"
  }
}
```

## Usage

```typescript
import { createClient, wrapResult } from '@hatchet/snips';

// Create a configured client
const client = createClient({
  baseUrl: 'https://api.example.com',
});

// Use the wrapResult utility to handle errors
const fetchData = async () => {
  const result = await wrapResult(fetch('https://api.example.com/data'));
  
  if (result.success) {
    return result.data;
  } else {
    console.error(result.error);
  }
};
```

## Development

```bash
# Install dependencies
pnpm install

# Build the package
pnpm build

# Watch for changes during development
pnpm dev

# Lint code
pnpm lint:check

# Fix linting issues
pnpm lint:fix
```

## Adding to the Library

When adding new functionality to this library, please follow these guidelines:

1. Add types to `src/types.ts`
2. Add utility functions to appropriate files in `src/`
3. Export public API from `src/index.ts`
4. Update documentation
5. Run tests if applicable 