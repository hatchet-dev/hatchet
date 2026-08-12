import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';
import { Document, FILENAME_REMAP } from './doc_types';
import { crawlDirectory } from './paths';
import { FRONTEND_DOCS_RELATIVE_PATH, TMP_GEN_PATH } from './shared';

// Core entrypoints that must always exist; feature clients are globbed dynamically so a
// new file in src/v1/client/features/ gets a docs page with zero config edits.
const CORE_ENTRYPOINTS = [
  './src/v1/client/client.ts',
  './src/v1/client/worker/context.ts',
  './src/v1/declaration.ts',
];

const FEATURES_DIR = './src/v1/client/features';

function collectEntryPoints(): string[] {
  const missing = CORE_ENTRYPOINTS.filter((f) => !fs.existsSync(f));
  if (missing.length) {
    throw new Error(
      `Core typedoc entrypoints are missing: ${missing.join(', ')}. ` +
        'If these files moved, update CORE_ENTRYPOINTS in sdks/typescript/docs/generate.ts.'
    );
  }

  const features = fs
    .readdirSync(FEATURES_DIR)
    .filter((f) => f.endsWith('.ts') && !f.endsWith('.test.ts') && f !== 'index.ts')
    .sort()
    .map((f) => `${FEATURES_DIR}/${f}`);

  return [...CORE_ENTRYPOINTS, ...features];
}

function rmrf(target: string) {
  if (fs.existsSync(target)) {
    fs.rmSync(target, { recursive: true, force: true });
  }
}

function dirTitle(dirName: string): string {
  return dirName.replace(/-/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

function subDirNames(documents: Document[]): string[] {
  return [...new Set(documents.map((d) => d.directory).filter(Boolean))]
    .map((d) => d.replace(/^\//, ''))
    .sort();
}

function writeJson(filePath: string, value: unknown) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, `${JSON.stringify(value, null, 2)}\n`, 'utf-8');
  console.log('Wrote', filePath);
}

// Fully regenerates the fumadocs meta.json for each generated subdirectory
// (e.g. feature-clients): pages sorted alphabetically.
function writeSubdirMetaJson(documents: Document[]) {
  for (const dirName of subDirNames(documents)) {
    const pages = documents
      .filter((d) => d.directory === `/${dirName}`)
      .map((d) => d.basename)
      .sort((a, b) => a.localeCompare(b));

    writeJson(path.join(FRONTEND_DOCS_RELATIVE_PATH, dirName, 'meta.json'), {
      pages,
      title: dirTitle(dirName),
    });
  }
}

const isSeparator = (page: string) => /^---.*---$/.test(page);

// Merges the section-level typescript/meta.json: preserves existing entry order and
// separator strings, drops entries whose page no longer exists, and appends newly
// emitted top-level pages/subdirectories. The parent reference/meta.json is never touched.
function mergeTopLevelMetaJson(documents: Document[]) {
  const metaPath = path.join(FRONTEND_DOCS_RELATIVE_PATH, 'meta.json');
  const valid = new Set([
    ...documents.filter((d) => !d.directory).map((d) => d.basename),
    ...subDirNames(documents),
  ]);

  const existing: { pages?: string[]; title?: string } = fs.existsSync(metaPath)
    ? JSON.parse(fs.readFileSync(metaPath, 'utf-8'))
    : {};

  const kept = (existing.pages ?? []).filter((p) => isSeparator(p) || valid.has(p));
  const appended = [...valid].filter((p) => !kept.includes(p)).sort();

  writeJson(metaPath, {
    ...existing,
    pages: [...kept, ...appended],
    title: existing.title ?? 'TypeScript SDK',
  });
}

// Backstop: every emitted .mdx must be reachable from a meta.json pages array,
// and every generated subdirectory must be listed in the top-level meta.json.
function assertAllPagesReachable(documents: Document[]) {
  const pagesOf = (dir: string): string[] =>
    JSON.parse(fs.readFileSync(path.join(dir, 'meta.json'), 'utf-8')).pages ?? [];

  const problems = documents
    .filter((d) => !pagesOf(path.dirname(d.mdxOutputPath)).includes(d.basename))
    .map((d) => `${d.mdxOutputPath} is not listed in its meta.json pages array`);

  const topPages = pagesOf(FRONTEND_DOCS_RELATIVE_PATH);
  problems.push(
    ...subDirNames(documents)
      .filter((dir) => !topPages.includes(dir))
      .map((dir) => `subdirectory "${dir}" is not listed in the top-level meta.json pages array`)
  );

  if (problems.length) {
    console.error('Docs generation failed: unreachable pages detected:');
    for (const p of problems) {
      console.error(`  - ${p}`);
    }
    process.exit(1);
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

  // Browsers resolve .mdx hrefs literally and 404 — link to the extensionless route.
  result = result.replace(/\(([^)\s]+)\.mdx(#[^)]*)?\)/g, '($1$2)');

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
  const files = [
    ...documents.map((d) => d.mdxOutputPath),
    path.join(FRONTEND_DOCS_RELATIVE_PATH, 'meta.json'),
    ...subDirNames(documents).map((dir) =>
      path.join(FRONTEND_DOCS_RELATIVE_PATH, dir, 'meta.json')
    ),
  ].sort();
  console.log('Running prettier on generated docs...');
  execSync(`npx prettier --write ${files.map((f) => `'${f}'`).join(' ')}`, { stdio: 'inherit' });
}

function run() {
  rmrf(TMP_GEN_PATH);

  try {
    const entryPoints = collectEntryPoints();
    console.log(`Running typedoc with ${entryPoints.length} entrypoints...`);
    execSync(`npx typedoc ${entryPoints.map((e) => `--entryPoints ${e}`).join(' ')}`, {
      stdio: 'inherit',
    });

    const documents = crawlDirectory(TMP_GEN_PATH).sort((a, b) =>
      a.mdxOutputPath.localeCompare(b.mdxOutputPath)
    );
    console.log(`Found ${documents.length} documents`);

    for (const doc of documents) {
      copyDoc(doc);
    }

    removeStaleMdx(documents);
    writeSubdirMetaJson(documents);
    mergeTopLevelMetaJson(documents);
    assertAllPagesReachable(documents);
    formatOutput(documents);
  } finally {
    rmrf(TMP_GEN_PATH);
  }
}

if (require.main === module) {
  run();
}
