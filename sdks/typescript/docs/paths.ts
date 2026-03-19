import fs from 'fs';
import path from 'path';
import { Document, documentFromPath } from './doc_types';

export function crawlDirectory(directory: string, onlyInclude: string[] = []): Document[] {
  return fs
    .readdirSync(directory, { withFileTypes: true })
    .filter((entry) => entry.isFile() && entry.name.endsWith('.mdx') && entry.name !== 'README.mdx')
    .map((entry) => documentFromPath(path.join(directory, entry.name)))
    .filter((doc) => !onlyInclude.length || onlyInclude.includes(doc.readableSourcePath));
}
