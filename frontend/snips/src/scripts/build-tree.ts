import { getConfig } from '../utils/config';
import fs from 'fs/promises';
import { Dirent } from 'fs';
import path from 'path';
import { clean, restore } from './clean';
import { colors } from '../utils/colors';

interface ProcessorResult {
  content: string;
  filename?: string;
}

interface Processor {
  (params: { path: string; name: string; content: string }): Promise<ProcessorResult>;
}

/**
 * Processes files in the source directory, preserving the directory structure
 * but transforming each file into a TypeScript file that exports a Snippet.
 */
export const processFiles = async (): Promise<string[]> => {
  const config = getConfig();
  const { SOURCE_DIRS, OUTPUT_DIR, IGNORE_LIST, PRESERVE_FILES } = config;

  console.log(`${colors.bright}${colors.blue}🚀 Starting snips processing...${colors.reset}`);
  console.log(`${colors.cyan}Source directories: ${SOURCE_DIRS.join(', ')}${colors.reset}`);
  console.log(`${colors.cyan}Output directory: ${OUTPUT_DIR}${colors.reset}`);

  // Handle case when no source directories are provided
  if (!SOURCE_DIRS || SOURCE_DIRS.length === 0) {
    console.log(`${colors.red}No source directories provided!${colors.reset}`);
    return [];
  }

  // Ensure output directory exists
  try {
    await fs.mkdir(OUTPUT_DIR, { recursive: true });
    console.log(`${colors.green}✓ Output directory created/verified: ${OUTPUT_DIR}${colors.reset}`);
  } catch (error) {
    console.error(
      `${colors.red}Error creating output directory ${OUTPUT_DIR}:${colors.reset}`,
      error,
    );
    throw error;
  }

  // Clean the output directory first
  const toRestore = await clean(OUTPUT_DIR, PRESERVE_FILES);

  // Process directories
  for (const sourceDir of SOURCE_DIRS) {
    console.log(`${colors.magenta}Processing directory: ${sourceDir}${colors.reset}`);
    // Recursively process the directory
    await processDirectory(sourceDir, OUTPUT_DIR, IGNORE_LIST);
  }

  // Restore the preserved files
  await restore(toRestore);

  console.log(`${colors.bright}${colors.green}✓ Processing complete!${colors.reset}`);

  // Return files from the first directory for testing purposes
  try {
    return await fs.readdir(SOURCE_DIRS[0]);
  } catch (error) {
    console.error(`${colors.red}Error reading directory ${SOURCE_DIRS[0]}:${colors.reset}`, error);
    throw error;
  }
};

/**
 * Ensures a directory exists, creating it if necessary
 */
const ensureDirectoryExists = async (dirPath: string): Promise<void> => {
  await fs.mkdir(dirPath, { recursive: true });
};

/**
 * Processes a single file, applying all processors and writing the result
 */
const processFile = async (
  sourcePath: string,
  outputPath: string,
  entry: Dirent,
  processors: Processor[],
): Promise<void> => {
  const content = await fs.readFile(sourcePath, 'utf-8');
  let currentContent = content;
  let currentOutputPath = outputPath;

  for (const processor of processors) {
    const result = await processor({
      path: sourcePath,
      name: entry.name,
      content: currentContent,
    });

    currentContent = result.content;

    if (result.filename) {
      const previousPath = currentOutputPath;
      currentOutputPath = path.join(path.dirname(currentOutputPath), result.filename);
      console.log(
        `${colors.yellow}  ⟳ Processor changed filename: ${path.basename(previousPath)} → ${result.filename}${colors.reset}`,
      );
    }
  }

  await ensureDirectoryExists(path.dirname(currentOutputPath));
  await fs.writeFile(currentOutputPath, currentContent, 'utf-8');
  console.log(`${colors.green}  ✓ Processed file written to: ${currentOutputPath}${colors.reset}`);
};

/**
 * Processes a single directory entry (file or subdirectory)
 */
const processEntry = async (
  entry: Dirent,
  sourcePath: string,
  outputDir: string,
  ignoreList: string[] | RegExp[],
  processors: Processor[],
): Promise<void> => {
  if (shouldIgnore(entry.name, ignoreList)) {
    console.log(`${colors.yellow}Ignoring: ${sourcePath}${colors.reset}`);
    return;
  }

  const targetPath = path.join(outputDir, entry.name);

  if (entry.isDirectory()) {
    console.log(`${colors.magenta}→ Processing subdirectory: ${sourcePath}${colors.reset}`);
    await processDirectory(sourcePath, targetPath, ignoreList);
  } else {
    console.log(`${colors.blue}→ Processing file: ${sourcePath}${colors.reset}`);
    await processFile(sourcePath, targetPath, entry, processors);
  }
};

/**
 * Recursively processes a directory and its contents
 */
export const processDirectory = async (
  sourceDir: string,
  outputDir: string,
  ignoreList: string[] | RegExp[],
): Promise<void> => {
  try {
    const { PROCESSORS } = getConfig();

    const entries = await fs.readdir(sourceDir, { withFileTypes: true });
    console.log(`${colors.cyan}Found ${entries.length} entries in ${sourceDir}${colors.reset}`);

    for (const entry of entries) {
      const sourcePath = path.join(sourceDir, entry.name);
      await processEntry(entry, sourcePath, outputDir, ignoreList, PROCESSORS);
    }
  } catch (error) {
    console.error(`${colors.red}Error processing directory ${sourceDir}:${colors.reset}`, error);
    throw error;
  }
};

/**
 * Checks if a file or directory should be ignored
 */
const shouldIgnore = (name: string, ignoreList: string[] | RegExp[]): boolean => {
  return ignoreList.some((pattern) => {
    if (pattern instanceof RegExp) {
      return pattern.test(name);
    }
    if (pattern.includes('*')) {
      // Convert glob pattern to regex
      const regexPattern = pattern.replace(/\./g, '\\.').replace(/\*/g, '.*');
      return new RegExp(`^${regexPattern}$`).test(name);
    }
    return name === pattern;
  });
};

processFiles();
