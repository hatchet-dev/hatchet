import { processFiles } from './build-tree';
import * as fs from 'fs/promises';
import * as path from 'path';

async function compareDirectories(actualDir: string, expectedDir: string) {
  const actualFiles = await fs.readdir(actualDir);
  const expectedFiles = await fs.readdir(expectedDir);

  // Check that both directories have exactly the same files
  expect(actualFiles.sort()).toEqual(expectedFiles.sort());

  for (const file of actualFiles) {
    const actualPath = path.join(actualDir, file);
    const expectedPath = path.join(expectedDir, file);

    const actualStat = await fs.stat(actualPath);
    const expectedStat = await fs.stat(expectedPath);

    if (actualStat.isDirectory() && expectedStat.isDirectory()) {
      await compareDirectories(actualPath, expectedPath);
    } else if (actualStat.isFile() && expectedStat.isFile()) {
      const actualContent = await fs.readFile(actualPath, 'utf8');
      const expectedContent = await fs.readFile(expectedPath, 'utf8');
      expect(actualContent).toEqual(expectedContent);
    } else {
      throw new Error(`Mismatched types for ${file}`);
    }
  }
}

describe('processFiles', () => {
  it('should process files correctly', async () => {
    await processFiles();

    const actualDir = path.join(__dirname, '../../out');
    const expectedDir = path.join(__dirname, '../../test_dir/expected');

    await compareDirectories(actualDir, expectedDir);
  });
});
