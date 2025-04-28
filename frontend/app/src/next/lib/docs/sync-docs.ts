#!/usr/bin/env node
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

// Get the directory paths
const dirname = path.dirname(fileURLToPath(import.meta.url));
const docsDir = path.resolve(dirname, '../../../../../docs/pages');
const generatedDir = path.resolve(dirname, 'generated');

// Make sure the generated directory exists
if (!fs.existsSync(generatedDir)) {
  fs.mkdirSync(generatedDir, { recursive: true });
}

// Keep track of found meta files for generating the index
const metaFiles: { importName: string; importPath: string }[] = [];

// Function to process a directory recursively
function processDirectory(dirPath: string, relativePath: string = '') {
  const entries = fs.readdirSync(dirPath, { withFileTypes: true });

  // First, find if there's a _meta.js file in this directory
  const metaFile = entries.find(
    (entry) => entry.isFile() && entry.name === '_meta.js',
  );

  if (metaFile) {
    // Determine the target directory path
    const targetDirPath = path.join(generatedDir, relativePath);

    // Create the target directory if it doesn't exist
    if (!fs.existsSync(targetDirPath)) {
      fs.mkdirSync(targetDirPath, { recursive: true });
    }

    // Read the meta file
    const metaFilePath = path.join(dirPath, metaFile.name);
    const metaContent = fs.readFileSync(metaFilePath, 'utf8');

    try {
      // Build a dictionary from the meta.js file
      const metaObject = parseMetaFile(metaContent);

      // Process the object to ensure all entries have title and href
      const processedObj = processMetaObject(metaObject, relativePath);

      // Convert to well-formatted string
      const formattedStr = formatObject(processedObj);

      // Create a TypeScript version with the proper format
      const tsContent = `// Generated from ${metaFilePath}
const meta = ${formattedStr};
export default meta;
`;

      // Write the TypeScript file
      const targetFilePath = path.join(targetDirPath, '_meta.ts');
      fs.writeFileSync(targetFilePath, tsContent, 'utf8');

      // Add to the list of meta files for the index
      let importName = relativePath.replace(/\//g, '') || 'root';

      // Handle nested paths by creating camelCase names
      if (importName !== 'root') {
        importName = importName
          .split('/')
          .map((part, index) => {
            if (index === 0) {
              return part;
            }
            return part.charAt(0).toUpperCase() + part.slice(1);
          })
          .join('');
      }

      metaFiles.push({
        importName: importName.replace(/-/g, '_'),
        importPath: `./${relativePath ? relativePath + '/' : ''}_meta`,
      });
    } catch (err) {
      console.error(`Error processing ${metaFilePath}:`, err);
      // If we can't process it, just use the original content with minimal transforms
      const tsContent = `// Generated from ${metaFilePath}
const meta = ${metaContent.replace('export default', '')};
export default meta;
`;
      const targetFilePath = path.join(targetDirPath, '_meta.ts');
      fs.writeFileSync(targetFilePath, tsContent, 'utf8');
    }
  }

  // Process subdirectories
  for (const entry of entries) {
    if (entry.isDirectory()) {
      processDirectory(
        path.join(dirPath, entry.name),
        relativePath ? path.join(relativePath, entry.name) : entry.name,
      );
    }
  }
}

// Function to parse a meta.js file into a dictionary
function parseMetaFile(content: string): Record<string, any> {
  // Remove the "export default" and trailing semicolon
  const objectStr = content
    .replace(/export\s+default\s*/, '')
    .trim()
    .replace(/;$/, '');

  // Parse the object structure
  const result: Record<string, any> = {};

  // Simple regex-based parser for the object structure
  // Extract key-value pairs from the object
  let depth = 0;
  let inString = false;
  let stringDelimiter = '';

  // Skip the first opening brace and last closing brace
  const objContent = objectStr
    .substring(objectStr.indexOf('{') + 1, objectStr.lastIndexOf('}'))
    .trim();

  // Split by commas that are not inside nested objects or strings
  const entries: string[] = [];
  let currentEntry = '';

  for (let i = 0; i < objContent.length; i++) {
    const char = objContent[i];

    if (char === '{') {
      depth++;
    }
    if (char === '}') {
      depth--;
    }

    if (char === '"' || char === "'") {
      if (!inString) {
        inString = true;
        stringDelimiter = char;
      } else if (char === stringDelimiter && objContent[i - 1] !== '\\') {
        inString = false;
      }
    }

    if (char === ',' && depth === 0 && !inString) {
      entries.push(currentEntry.trim());
      currentEntry = '';
    } else {
      currentEntry += char;
    }
  }

  if (currentEntry.trim()) {
    entries.push(currentEntry.trim());
  }

  // Process each entry
  for (const entry of entries) {
    // Split by the first colon not in a string
    let colonIndex = -1;
    inString = false;
    stringDelimiter = '';

    for (let i = 0; i < entry.length; i++) {
      const char = entry[i];

      if (char === '"' || char === "'") {
        if (!inString) {
          inString = true;
          stringDelimiter = char;
        } else if (char === stringDelimiter && entry[i - 1] !== '\\') {
          inString = false;
        }
      }

      if (char === ':' && !inString && colonIndex === -1) {
        colonIndex = i;
        break;
      }
    }

    if (colonIndex === -1) {
      continue;
    }

    const keyPart = entry.substring(0, colonIndex).trim();
    const valuePart = entry.substring(colonIndex + 1).trim();

    // Extract the key (remove quotes if present)
    let key = keyPart;
    if (
      (key.startsWith('"') && key.endsWith('"')) ||
      (key.startsWith("'") && key.endsWith("'"))
    ) {
      key = key.substring(1, key.length - 1);
    }

    // Parse the value
    let value: any;

    if (valuePart === 'true') {
      value = true;
    } else if (valuePart === 'false') {
      value = false;
    } else if (valuePart === 'null') {
      value = null;
    } else if (valuePart === 'undefined') {
      value = undefined;
    } else if (valuePart.startsWith('{') && valuePart.endsWith('}')) {
      // Nested object - recursively parse
      value = parseMetaFile(`export default ${valuePart}`);
    } else if (
      (valuePart.startsWith('"') && valuePart.endsWith('"')) ||
      (valuePart.startsWith("'") && valuePart.endsWith("'"))
    ) {
      // String value
      value = valuePart.substring(1, valuePart.length - 1);
    } else if (!isNaN(Number(valuePart))) {
      // Number value
      value = Number(valuePart);
    } else {
      // Unknown/complex value - keep as string
      value = valuePart;
    }

    result[key] = value;
  }

  return result;
}

// Function to format an object as a string
function formatObject(obj: Record<string, any>, indent = 0): string {
  const spaces = ' '.repeat(indent);
  let result = '{\n';

  for (const [key, value] of Object.entries(obj)) {
    // Format the key - add quotes for keys with special characters
    const formattedKey = /^[a-zA-Z0-9_]+$/.test(key) ? key : `'${key}'`;

    // Format the value based on type
    let formattedValue: string;

    if (value === null) {
      formattedValue = 'null';
    } else if (typeof value === 'object') {
      formattedValue = formatObject(value, indent + 2);
    } else if (typeof value === 'string') {
      formattedValue = `'${value.replace(/'/g, "\\'")}'`;
    } else {
      formattedValue = String(value);
    }

    result += `${spaces}  ${formattedKey}: ${formattedValue},\n`;
  }

  if (result.endsWith(',\n')) {
    result = result.slice(0, -2) + '\n';
  }

  result += `${spaces}}`;
  return result;
}

// Function to recursively process the meta object
function processMetaObject(
  obj: Record<string, any>,
  relativePath: string,
): Record<string, any> {
  const result: Record<string, any> = {};

  // Process each key in the object
  for (const [key, value] of Object.entries(obj)) {
    // Sanitize the key - replace hyphens with underscores, except for --prefixed keys
    const sanitizedKey = key.startsWith('--') ? key : key.replace(/-/g, '_');

    // Skip --prefixed entries (but preserve them)
    if (key.startsWith('--')) {
      result[sanitizedKey] = value;
      continue;
    }

    if (typeof value === 'string') {
      // Convert string values to objects with title and href
      result[sanitizedKey] = {
        title: value,
        href:
          key === 'index'
            ? `/${relativePath ? relativePath + '/' : ''}`
            : `/${relativePath ? relativePath + '/' : ''}${key}`, // Keep original key for href
      };
    } else if (typeof value === 'object' && value !== null) {
      // Already an object, make sure it has href if it has title
      if (value.title && !value.href && !value.type) {
        result[sanitizedKey] = {
          ...value,
          href: `/${relativePath ? relativePath + '/' : ''}${key}`, // Keep original key for href
        };
      } else {
        // Just keep the original object
        result[sanitizedKey] = value;
      }
    } else {
      // For any other type, just pass it through
      result[sanitizedKey] = value;
    }
  }

  return result;
}

// Start processing from the root docs directory
processDirectory(docsDir);

// Generate the index.ts file
const indexContent = `// Generated index file for meta-data
${metaFiles.map((file) => `import ${file.importName} from '${file.importPath}';`).join('\n')}

export { ${metaFiles.map((file) => file.importName).join(', ')} };
`;

// Write the index file
fs.writeFileSync(path.join(generatedDir, 'index.ts'), indexContent, 'utf8');

console.log(`Generated ${metaFiles.length} meta files in ${generatedDir}`);
