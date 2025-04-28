import fs from 'fs/promises';
import path from 'path';
import { colors } from '../utils/colors';
import { tmpdir } from 'os';
/**
 * Preserved file structure for restoring files later
 */
interface PreservedFiles {
  files: string[];
  tempPath: string;
}

/**
 * Cleans the output directory by temporarily preserving important files,
 * removing everything, then returning info needed to restore the preserved files later
 */
export const clean = async (
  dir: string,
  preserveFiles: string[] | RegExp[],
  tmpDir?: string,
): Promise<PreservedFiles> => {
  if (!tmpDir) {
    tmpDir = path.join(tmpdir(), `snips-${Date.now()}`);
    await fs.mkdir(tmpDir, { recursive: true });
  }

  const toRestore: PreservedFiles['files'] = [];

  // Get all entries in the directory
  const entries = await fs.readdir(dir, { withFileTypes: true });
  for (const entry of entries) {
    const filePath = path.join(dir, entry.name);
    const stats = await fs.stat(filePath);
    if (stats.isDirectory()) {
      // Recursively clean the directory
      const results = await clean(filePath, preserveFiles, tmpDir);
      toRestore.push(...results.files);
    } else {
      console.log(`${colors.cyan}Checking file: ${entry.name}${colors.reset}`);
      // Check if the file should be preserved
      if (
        preserveFiles.some((p) => (typeof p === 'string' ? p === entry.name : p.test(entry.name)))
      ) {
        console.log(`${colors.cyan}Preserving file: ${entry.name}${colors.reset}`);
        // Create the temporary directory structure
        const tempFilePath = path.join(tmpDir, filePath);
        await fs.mkdir(path.dirname(tempFilePath), { recursive: true });
        // Copy the file to temporary storage
        await fs.copyFile(filePath, tempFilePath);
        toRestore.push(filePath);
      }
    }
  }

  // After preserving files, remove the directory and its contents
  if (dir !== tmpDir) {
    try {
      await fs.rm(dir, { recursive: true, force: true });
      console.log(`${colors.yellow}Removed directory: ${dir}${colors.reset}`);
    } catch (error) {
      console.error(`${colors.red}Error removing directory ${dir}:${colors.reset}`, error);
    }
  }

  return {
    files: toRestore,
    tempPath: tmpDir,
  };
};

/**
 * Restores preserved files from temporary directory to output directory
 */
export const restore = async ({ files, tempPath }: PreservedFiles): Promise<void> => {
  // Then restore all files
  for (const file of files) {
    try {
      const tempFilePath = path.join(tempPath, file);
      const dirPath = path.dirname(file);
      await fs.mkdir(dirPath, { recursive: true });
      await fs.copyFile(tempFilePath, file);
      console.log(`${colors.green}Restored preserved file: ${file}${colors.reset}`);
    } catch (error) {
      console.error(`${colors.red}Error restoring file: ${file}${colors.reset}`, error);
    }
  }
};
