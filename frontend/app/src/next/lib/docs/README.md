# Docs Generation Utility

This utility converts the `_meta.js` files from the Nextra documentation in `/frontend/docs` into TypeScript files that can be imported and used in the main app.

## How it works

The script:
1. Finds all `_meta.js` files in the `/frontend/docs/pages` directory and subdirectories
2. Converts them to TypeScript files with proper exports
3. Writes them to the `generated` directory
4. Creates an `index.ts` file that exports all the metadata objects

## Usage

Run the script with:

```bash
pnpm run generate:docs
```

## Generated Output

The script generates:
- TypeScript versions of all `_meta.js` files in the `/frontend/docs/pages` directory
- An `index.ts` file that imports and exports all metadata objects
- Directory structure that matches the original structure in `/frontend/docs/pages`

Import the generated docs metadata in your application:

```typescript
import * as docs from 'src/next/lib/docs';
// or
import { root, blog, home } from 'src/next/lib/docs';
```
