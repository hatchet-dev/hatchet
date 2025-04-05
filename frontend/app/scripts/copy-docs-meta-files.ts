// This script copies _meta.js files from the docs directory to the app directory
import * as fs from 'node:fs';
import * as path from 'node:path';
import { fileURLToPath } from 'node:url';

const currentFilename = fileURLToPath(import.meta.url);
const currentDirname = path.dirname(currentFilename);
const rootDir = path.resolve(currentDirname, '..');
const docsDir = path.resolve(rootDir, '..', 'docs');
const metaDataDir = path.resolve(rootDir, 'src', 'docs-meta-data');
const outputDir = path.resolve(metaDataDir, 'generated');

interface MetaFile {
  sourcePath: string;
  relativePath: string;
  fileName: string;
}

// Clean up the generated directory if it exists
if (fs.existsSync(outputDir)) {
  console.log(`Cleaning up existing generated directory: ${outputDir}`);
  fs.rmSync(outputDir, { recursive: true, force: true });
}

// Create the output directory
console.log(`Creating output directory: ${outputDir}`);
fs.mkdirSync(outputDir, { recursive: true });

// Function to find all _meta.js files
function findMetaFiles(dir: string, baseDir = dir): MetaFile[] {
  const results: MetaFile[] = [];
  const files = fs.readdirSync(dir);

  for (const file of files) {
    const filePath = path.join(dir, file);
    const stat = fs.statSync(filePath);

    if (stat.isDirectory()) {
      results.push(...findMetaFiles(filePath, baseDir));
    } else if (file === '_meta.js') {
      // Get relative path from base directory
      const relativePath = path.relative(baseDir, dir);
      results.push({
        sourcePath: filePath,
        relativePath,
        fileName: file,
      });
    }
  }

  return results;
}

// Function to convert simple string entries to objects with title and href
function processMetaContent(content: string, relativePath: string): string {
  try {
    // Extract the object part
    const metaMatch = content.match(/export\s+default\s+(\{[\s\S]*\});?/);
    if (!metaMatch || !metaMatch[1]) {
      return content;
    }

    // Try to parse, but if it fails, return the original content
    let metaObj;
    try {
      // Using eval is not ideal, but necessary to parse JS object syntax that isn't valid JSON
      // eslint-disable-next-line no-eval
      metaObj = eval(`(${metaMatch[1]})`);
    } catch (e) {
      console.warn(`Failed to parse meta content for ${relativePath}: ${e}`);
      return content;
    }

    // Calculate the parent path for hrefs
    const parentPath = relativePath ? `/${relativePath}` : '';

    // Process each entry
    for (const [key, value] of Object.entries(metaObj)) {
      // Skip entries that start with -- as they are separators
      if (key.startsWith('--')) {
        continue;
      }

      if (typeof value === 'string') {
        // Convert strings to objects with title and href
        metaObj[key] = {
          title: value,
          href: `${parentPath}/${key === 'index' ? '' : key}`,
        };
      } else if (
        typeof value === 'object' &&
        value !== null &&
        !('href' in value) &&
        !('type' in value)
      ) {
        // Add href to objects that don't have one already and aren't separators or other special types
        if ('title' in value) {
          metaObj[key] = {
            ...value,
            href: `${parentPath}/${key}`,
          };
        }
      }
    }

    // Convert back to string format
    return content.replace(metaMatch[1], JSON.stringify(metaObj, null, 2));
  } catch (error) {
    console.warn(`Error processing meta content for ${relativePath}: ${error}`);
    return content;
  }
}

// Find all _meta.js files in the docs/pages directory
const metaFiles = findMetaFiles(path.join(docsDir, 'pages'));

console.log(`Found ${metaFiles.length} _meta.js files in docs directory`);

// Copy each file to the corresponding location in the app directory
metaFiles.forEach(({ sourcePath, relativePath, fileName }) => {
  // Create the destination directory
  const destDir = path.join(outputDir, relativePath);
  if (!fs.existsSync(destDir)) {
    fs.mkdirSync(destDir, { recursive: true });
  }

  // Read the source file
  const fileContent = fs.readFileSync(sourcePath, 'utf8');

  // Process the content to convert string entries to objects
  const processedContent = processMetaContent(fileContent, relativePath);

  // Convert to TypeScript by adding a type and converting to const export
  const tsContent = `// Generated from ${sourcePath}
const meta = ${processedContent.replace('export default ', '')};

export default meta;
`;

  // Write to the destination file with .ts extension
  const destPath = path.join(destDir, fileName.replace('.js', '.ts'));
  fs.writeFileSync(destPath, tsContent);

  console.log(`Copied and converted ${sourcePath} to ${destPath}`);
});

console.log('Finished copying and converting _meta.js files');

// Create an index.ts file that exports all the meta files
const exportNames = metaFiles.map(({ relativePath }) => {
  return relativePath
    ? toCamelCase(relativePath.replace(/\//g, '_').replace(/-/g, '_'))
    : 'root';
});

// Function to convert snake_case to camelCase
function toCamelCase(str: string): string {
  return str.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase());
}

const imports = metaFiles
  .map(({ relativePath, fileName }, index) => {
    // Remove .js extension from the import path but don't add .ts (TypeScript will resolve it)
    const importPath = `./${relativePath ? relativePath + '/' : ''}${fileName.replace('.js', '')}`;
    const exportName = exportNames[index];
    return `import ${exportName} from '${importPath}';`;
  })
  .join('\n');

const exports = `export { ${exportNames.join(', ')} };`;

const indexContent = `// Generated index file for meta-data
${imports}

${exports}
`;

fs.writeFileSync(path.join(outputDir, 'index.ts'), indexContent);
console.log('Created index.ts file');
