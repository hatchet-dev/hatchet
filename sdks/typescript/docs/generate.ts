import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';
import { Document } from './doc_types';
import { crawlDirectory } from './paths';
import { TMP_GEN_PATH } from './shared';

function rmrf(target: string) {
  if (fs.existsSync(target)) {
    fs.rmSync(target, { recursive: true, force: true });
  }
}

function metaEntry(key: string, title: string): string {
  return `  "${key}": {
    title: "${title}",
    theme: {
      toc: true,
    },
  },`;
}

function generateMetaJs(docs: Document[], subDirs: string[]): string {
  const docEntries = docs.map((d) => d.metaJsEntry);
  const subDirEntries = subDirs.map((dir) => {
    const name = dir.replace(/^\//, '');
    const title = name.replace(/-/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
    return metaEntry(name, title);
  });

  const all = [...docEntries, ...subDirEntries].sort((a, b) => {
    const keyA = a.trim().split(':')[0].replace(/"/g, '').toLowerCase();
    const keyB = b.trim().split(':')[0].replace(/"/g, '').toLowerCase();
    return keyA.localeCompare(keyB);
  });

  return `export default {\n${all.join('\n\n')}\n};\n`;
}

function updateMetaJs(documents: Document[]) {
  const metaJsPaths = new Set(documents.map((d) => d.mdxOutputMetaJsPath));

  for (const metaJsPath of metaJsPaths) {
    const relevantDocs = documents.filter((d) => d.mdxOutputMetaJsPath === metaJsPath);
    const thisDir = relevantDocs[0].directory;

    const subDirs = [
      ...new Set(
        documents
          .map((d) => d.directory)
          .filter((dir) => dir !== thisDir && dir.startsWith(thisDir) && dir.split('/').length === thisDir.split('/').filter(Boolean).length + 2)
      ),
    ];

    const meta = generateMetaJs(relevantDocs, subDirs);

    fs.mkdirSync(path.dirname(metaJsPath), { recursive: true });
    fs.writeFileSync(metaJsPath, meta, 'utf-8');
    console.log('Wrote', metaJsPath);
  }
}

function fixBrokenFeatureAnchorLinks(content: string): string {
  return content.replace(/\(client\.features\.([^)\s#]+\.mdx)/g, '(feature-clients/$1');
}

function copyDoc(document: Document) {
  const content = fixBrokenFeatureAnchorLinks(fs.readFileSync(document.sourcePath, 'utf-8'));
  fs.mkdirSync(path.dirname(document.mdxOutputPath), { recursive: true });
  fs.writeFileSync(document.mdxOutputPath, content, 'utf-8');
  console.log('Wrote', document.mdxOutputPath);
}

function run() {
  rmrf(TMP_GEN_PATH);

  try {
    console.log('Running typedoc...');
    execSync('npx typedoc', { stdio: 'inherit' });

    const documents = crawlDirectory(TMP_GEN_PATH);
    console.log(`Found ${documents.length} documents`);

    for (const doc of documents) {
      copyDoc(doc);
    }

    updateMetaJs(documents);

    console.log('Running prettier on frontend docs...');
    execSync('pnpm lint:fix', {
      stdio: 'inherit',
      cwd: path.resolve(process.cwd(), '../../frontend/docs'),
      shell: '/bin/bash',
    });
  } finally {
    rmrf(TMP_GEN_PATH);
  }
}

if (require.main === module) {
  run();
}
