import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';
import { Document, FILENAME_REMAP } from './doc_types';
import { crawlDirectory } from './paths';
import { FRONTEND_DOCS_RELATIVE_PATH, TMP_GEN_PATH } from './shared';

function rmrf(target: string) {
  if (fs.existsSync(target)) {
    fs.rmSync(target, { recursive: true, force: true });
  }
}

function dirTitle(dirName: string): string {
  return dirName.replace(/-/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

// Writes a fumadocs meta.json for each generated subdirectory (e.g. feature-clients).
// The top-level typescript/meta.json is hand-maintained and intentionally not touched.
function writeSubdirMetaJson(documents: Document[]) {
  const subDirs = [...new Set(documents.map((d) => d.directory).filter(Boolean))].sort();

  for (const dir of subDirs) {
    const dirName = dir.replace(/^\//, '');
    const pages = documents
      .filter((d) => d.directory === dir)
      .map((d) => d.basename)
      .sort((a, b) => a.localeCompare(b));

    const meta = { pages, title: dirTitle(dirName) };
    const metaPath = path.join(FRONTEND_DOCS_RELATIVE_PATH, dirName, 'meta.json');

    fs.mkdirSync(path.dirname(metaPath), { recursive: true });
    fs.writeFileSync(metaPath, `${JSON.stringify(meta, null, 2)}\n`, 'utf-8');
    console.log('Wrote', metaPath);
  }
}

function fixLinks(content: string, document: Document): string {
  const inFeatureClients = document.directory === '/feature-clients';

  // typedoc flattens feature client modules to client.features.<name>.mdx; point links
  // at the feature-clients/ directory (or the sibling file when already inside it).
  let result = content.replace(/\(client\.features\.([^)\s#]+\.mdx)/g, (_m, leaf) =>
    inFeatureClients ? `(${leaf}` : `(feature-clients/${leaf}`
  );

  // Rewrite links to renamed top-level files (e.g. Runnables.mdx -> runnables.mdx).
  for (const [from, to] of Object.entries(FILENAME_REMAP)) {
    const target = inFeatureClients ? `../${to}` : to;
    result = result.split(`(${from}`).join(`(${target}`);
  }

  return result;
}

function withFrontmatter(content: string, document: Document): string {
  if (content.startsWith('---\n')) {
    return content;
  }
  return `---\ntitle: "${document.title}"\n---\n\n${content}`;
}

function copyDoc(document: Document) {
  const raw = fs.readFileSync(document.sourcePath, 'utf-8');
  const content = withFrontmatter(fixLinks(raw, document), document);
  fs.mkdirSync(path.dirname(document.mdxOutputPath), { recursive: true });
  fs.writeFileSync(document.mdxOutputPath, content, 'utf-8');
  console.log('Wrote', document.mdxOutputPath);
}

// The generator owns every .mdx in the output tree: remove any it did not emit.
function removeStaleMdx(documents: Document[]) {
  const emitted = new Set(documents.map((d) => path.resolve(d.mdxOutputPath)));

  const walk = (dir: string) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const full = path.join(dir, entry.name);
      if (entry.isDirectory()) {
        walk(full);
      } else if (
        (entry.name.endsWith('.mdx') && !emitted.has(path.resolve(full))) ||
        entry.name === '_meta.js'
      ) {
        fs.rmSync(full);
        console.log('Removed stale', full);
      }
    }
  };

  if (fs.existsSync(FRONTEND_DOCS_RELATIVE_PATH)) {
    walk(FRONTEND_DOCS_RELATIVE_PATH);
  }
}

function formatOutput(documents: Document[]) {
  const files = documents.map((d) => d.mdxOutputPath).sort();
  console.log('Running prettier on generated docs...');
  execSync(`npx prettier --write ${files.map((f) => `'${f}'`).join(' ')}`, { stdio: 'inherit' });
}

function run() {
  rmrf(TMP_GEN_PATH);

  try {
    console.log('Running typedoc...');
    execSync('npx typedoc', { stdio: 'inherit' });

    const documents = crawlDirectory(TMP_GEN_PATH).sort((a, b) =>
      a.mdxOutputPath.localeCompare(b.mdxOutputPath)
    );
    console.log(`Found ${documents.length} documents`);

    for (const doc of documents) {
      copyDoc(doc);
    }

    removeStaleMdx(documents);
    writeSubdirMetaJson(documents);
    formatOutput(documents);
  } finally {
    rmrf(TMP_GEN_PATH);
  }
}

if (require.main === module) {
  run();
}
