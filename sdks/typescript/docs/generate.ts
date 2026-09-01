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
  './src/v1/embedded.ts',
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

// Shared mapping pairing feature-client concepts with descriptions, per-language page
// slugs, and user-guide links. Hand-maintained; consumed by all four SDK doc generators.
const REFERENCE_MAP_PATH = '../../frontend/docs/reference-map.json';
const CONTENT_DOCS_PATH = '../../frontend/docs/content/docs';
const LANG = 'typescript';

interface RefMapCorePage {
  title: string;
  description: string;
}

interface RefMapFeature {
  title: string;
  description: string;
  guide: string | null;
  guideTitle: string | null;
  slugs: Record<string, string>;
}

interface RefMap {
  corePages: Record<string, RefMapCorePage>;
  featureClients: Record<string, RefMapFeature>;
}

function guidePageExists(guide: string): boolean {
  const rel = guide.replace(/^\//, '');
  return (
    fs.existsSync(path.join(CONTENT_DOCS_PATH, `${rel}.mdx`)) ||
    fs.existsSync(path.join(CONTENT_DOCS_PATH, rel, 'index.mdx'))
  );
}

// Renders the TypeScript SDK overview page (index.mdx): a link map over the core pages
// and a feature-clients table cross-linked to the user guide. Hard-fails when an emitted
// feature-client page has no reference-map.json entry, when an entry's typescript slug
// matches no emitted page (stale entry), or when a guide link points at a missing page.
function renderIndexPage(documents: Document[]): string {
  const map: RefMap = JSON.parse(fs.readFileSync(REFERENCE_MAP_PATH, 'utf-8'));

  const slugToConcept = new Map<string, string>();
  for (const [concept, feature] of Object.entries(map.featureClients)) {
    const slug = feature.slugs[LANG];
    if (!slug) {
      continue;
    }
    if (slugToConcept.has(slug)) {
      throw new Error(
        `reference-map.json: ${LANG} slug "${slug}" claimed by both "${slugToConcept.get(slug)}" and "${concept}"`
      );
    }
    slugToConcept.set(slug, concept);
  }

  const featureDocs = documents.filter((d) => d.directory === '/feature-clients');
  const emitted = new Set(featureDocs.map((d) => d.basename));
  for (const doc of featureDocs) {
    if (!slugToConcept.has(doc.basename)) {
      throw new Error(
        `reference-map.json has no featureClients entry with slugs.${LANG} = "${doc.basename}"; add one for the ${doc.title} client`
      );
    }
  }
  for (const [slug, concept] of slugToConcept) {
    if (!emitted.has(slug)) {
      throw new Error(
        `reference-map.json entry "${concept}" lists stale ${LANG} slug "${slug}": no such feature-client page is emitted`
      );
    }
  }

  const coreLines = documents
    .filter((d) => !d.directory)
    .map((d) => {
      const core = map.corePages[d.basename];
      if (!core) {
        throw new Error(`reference-map.json has no corePages entry for emitted page "${d.basename}"`);
      }
      return `- [${core.title}](/reference/${LANG}/${d.basename}): ${core.description}`;
    });

  const rows = featureDocs.map((doc) => {
    const feature = map.featureClients[slugToConcept.get(doc.basename)!];
    let guide = '';
    if (feature.guide) {
      if (!guidePageExists(feature.guide)) {
        throw new Error(
          `reference-map.json: guide "${feature.guide}" for "${feature.title}" does not exist under frontend/docs/content/docs`
        );
      }
      guide = `[${feature.guideTitle ?? feature.guide}](${feature.guide})`;
    }
    const name = `[${feature.title}](/reference/${LANG}/feature-clients/${doc.basename})`;
    return `| ${name} | ${feature.description} | ${guide} |`;
  });

  return [
    '---',
    'title: "Overview"',
    '---',
    '',
    '# TypeScript SDK',
    '',
    'This is the generated API reference for the Hatchet TypeScript SDK. For concepts and guides, see the [user guide](/v1).',
    '',
    '## Core pages',
    '',
    ...coreLines,
    '',
    '## Feature clients',
    '',
    'Feature clients are available as properties on the [client](/reference/typescript/client), and each covers one area of the Hatchet API. The Guide column links to the user guide page for the feature.',
    '',
    '| Client | Description | Guide |',
    '| ------ | ----------- | ----- |',
    ...rows,
    '',
  ].join('\n');
}

function writeIndexPage(documents: Document[]): Document {
  const indexDoc: Document = {
    sourcePath: '',
    readableSourcePath: 'index.mdx (generated overview page)',
    mdxOutputPath: path.join(FRONTEND_DOCS_RELATIVE_PATH, 'index.mdx'),
    isIndex: true,
    directory: '',
    basename: 'index',
    title: 'TypeScript SDK',
  };
  fs.writeFileSync(indexDoc.mdxOutputPath, renderIndexPage(documents), 'utf-8');
  console.log('Wrote', indexDoc.mdxOutputPath);
  return indexDoc;
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

function fixLinks(content: string): string {
  // typedoc flattens feature client modules to client.features.<name>.mdx; point links
  // at the feature-clients/ directory. Links are absolute so pages resolve correctly
  // when served from redirect or trailing-slash URLs.
  let result = content.replace(
    /\(client\.features\.([^)\s#]+\.mdx)/g,
    (_m, leaf) => `(/reference/typescript/feature-clients/${leaf}`
  );

  // Rewrite links to renamed top-level files (e.g. Runnables.mdx -> runnables.mdx).
  for (const [from, to] of Object.entries(FILENAME_REMAP)) {
    result = result.split(`(${from}`).join(`(/reference/typescript/${to}`);
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
  const content = withFrontmatter(fixLinks(raw), document);
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

    documents.push(writeIndexPage(documents));

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
