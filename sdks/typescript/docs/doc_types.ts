import path from 'path';
import { FRONTEND_DOCS_RELATIVE_PATH, MDX_EXTENSION } from './shared';

export interface Document {
  sourcePath: string;
  readableSourcePath: string;
  mdxOutputPath: string;
  mdxOutputMetaJsPath: string;
  isIndex: boolean;
  directory: string;
  basename: string;
  title: string;
  metaJsEntry: string;
}

const FILENAME_REMAP: Record<string, string> = {
  'Hatchet-TypeScript-SDK-Reference.mdx': 'client.mdx',
};

function remapFilename(filename: string): { outRelative: string; basename: string; directory: string } {
  const remapped = FILENAME_REMAP[filename] ?? filename;

  if (remapped.startsWith('client.features.')) {
    const leafName = remapped.replace('client.features.', '');
    return {
      outRelative: 'feature-clients/' + leafName,
      basename: path.basename(leafName, MDX_EXTENSION),
      directory: '/feature-clients',
    };
  }

  return {
    outRelative: remapped,
    basename: path.basename(remapped, MDX_EXTENSION),
    directory: '',
  };
}

function toTitle(basename: string): string {
  return basename
    .replace(/[-_.]/g, ' ')
    .replace(/[^0-9a-zA-Z ]+/g, '')
    .trim()
    .replace(/\s+/g, ' ')
    .split(' ')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
    .join(' ');
}


export function documentFromPath(filePath: string): Document {
  const filename = path.basename(filePath);
  const { outRelative, basename, directory } = remapFilename(filename);

  const title = toTitle(basename);
  const outFull = path.join(FRONTEND_DOCS_RELATIVE_PATH, outRelative);
  const outDir = path.dirname(outFull);

  const metaJsEntry = `  "${basename}": {
    title: "${title}",
    theme: {
      toc: true,
    },
  },`;

  return {
    directory,
    basename,
    title,
    metaJsEntry,
    sourcePath: filePath,
    readableSourcePath: filename,
    mdxOutputPath: outFull,
    mdxOutputMetaJsPath: path.join(outDir, '_meta.js'),
    isIndex: basename === 'index' || basename === 'README',
  };
}
