import { getConfig } from '../utils/config';
import fs from 'fs/promises';
import { Dirent } from 'fs';
import path from 'path';
import { clean, restore } from './clean-build';
import { colors } from '../utils/colors';
import { Processor } from '@/processors/processor.interface';

/**
 * Processes files in the source directory, preserving the directory structure
 * but transforming each file into a TypeScript file that exports a Snippet.
 */
export const processFiles = async (): Promise<string[]> => {
  const config = getConfig();
  const startTime = Date.now();
  const { SOURCE_DIRS, OUTPUT_DIR, IGNORE_LIST, PRESERVE_FILES } = config;

  console.log(`${colors.bright}${colors.blue}🚀 Starting snips processing...${colors.reset}`);
  console.log(
    `${colors.cyan}Source directories: ${Object.keys(SOURCE_DIRS).join(', ')}${colors.reset}`,
  );
  console.log(`${colors.cyan}Output directory: ${OUTPUT_DIR}${colors.reset}`);

  // Handle case when no source directories are provided
  if (!SOURCE_DIRS || Object.keys(SOURCE_DIRS).length === 0) {
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
  for (const [language, sourceDir] of Object.entries(SOURCE_DIRS)) {
    console.log(`${colors.magenta}Processing directory: ${sourceDir}${colors.reset}`);
    // Recursively process the directory
    await processDirectory(sourceDir, path.join(OUTPUT_DIR, language), IGNORE_LIST);
  }

  // Restore the preserved files
  await restore(toRestore);

  // Process all directories with all processors recursively
  const { PROCESSORS } = config;
  for (const processor of PROCESSORS) {
    console.log(
      `${colors.magenta}Running directory processor recursively: ${processor.constructor.name}${colors.reset}`,
    );
    await processFinalDirectoryRecursively(OUTPUT_DIR, processor);
  }

  const endTime = Date.now();
  const duration = (endTime - startTime) / 1000;
  console.log(
    `${colors.bright}${colors.green}✓ Processing complete in ${duration} seconds!${colors.reset}`,
  );

  // Return files from the first directory for testing purposes
  try {
    return await fs.readdir(Object.values(SOURCE_DIRS)[0]);
  } catch (error) {
    console.error(
      `${colors.red}Error reading directory ${Object.values(SOURCE_DIRS)[0]}:${colors.reset}`,
      error,
    );
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

  processors.forEach(async (processor) => {
    const results = await processor.processFile({
      path: outputPath,
      name: entry.name,
      content: content,
    });

    await Promise.all(
      results.map(async (result) => {
        const previousPath = outputPath;

        const previousPathParts = previousPath.split('/');
        let currentOutputPath = path.join(
          previousPathParts[0],
          result.outDir || '',
          ...previousPathParts.slice(1),
        );

        if (result.filename) {
          const previousPath = currentOutputPath;
          currentOutputPath = path.join(path.dirname(currentOutputPath), result.filename);
          console.log(
            `${colors.yellow}  ⟳ Processor changed filename: ${path.basename(previousPath)} → ${result.filename}${colors.reset}`,
          );
        }

        await ensureDirectoryExists(path.dirname(currentOutputPath));
        await fs.writeFile(currentOutputPath, result.content, 'utf-8');

        console.log(
          `${colors.green}  ✓ Processed file written to: ${currentOutputPath}${colors.reset}`,
        );
      }),
    );
  });
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

    await Promise.all(
      entries.map(async (entry) => {
        const sourcePath = path.join(sourceDir, entry.name);
        await processEntry(entry, sourcePath, outputDir, ignoreList, PROCESSORS);
      }),
    );
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

/**
 * Recursively processes a directory and all its subdirectories with a given processor
 */
const processFinalDirectoryRecursively = async (
  dirPath: string,
  processor: Processor,
): Promise<void> => {
  const entries = await fs.readdir(dirPath, { withFileTypes: true });

  // Process the current directory
  await processor.processDirectory({ dir: dirPath });

  // Process all subdirectories
  for (const entry of entries) {
    if (entry.isDirectory()) {
      const subDirPath = path.join(dirPath, entry.name);
      await processFinalDirectoryRecursively(subDirPath, processor);
    }
  }
};

processFiles();
