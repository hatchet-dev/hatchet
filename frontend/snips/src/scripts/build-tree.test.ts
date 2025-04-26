import fs from 'fs/promises';
import path from 'path';
import os from 'os';
import { getConfig, Config } from '@/utils/config';
import { processFiles } from './build-tree';

// Mock the config module
jest.mock('@/utils/config');

let tempDir: string;
let testSourceDir: string;

// Simple test configuration
const CONFIG: Config = {
  SOURCE_DIRS: [],
  OUTPUT_DIR: '',
  IGNORE_LIST: [],
  PRESERVE_FILES: [],
  REPLACEMENTS: [],
  REMOVAL_PATTERNS: [],
  PROCESSORS: [],
};

describe('processFiles', () => {
  beforeEach(async () => {
    jest.clearAllMocks();

    // Create temporary test directories
    tempDir = await fs.mkdtemp(path.join(os.tmpdir(), 'build-tree-test-'));
    testSourceDir = path.join(tempDir, 'source');
    const testOutputDir = path.join(tempDir, 'output');

    // Create source directory
    await fs.mkdir(testSourceDir, { recursive: true });

    // Update config with temp directories
    CONFIG.SOURCE_DIRS = [testSourceDir];
    CONFIG.OUTPUT_DIR = testOutputDir;

    (getConfig as jest.Mock).mockReturnValue(CONFIG);
  });

  afterEach(async () => {
    // Clean up temporary directory
    await fs.rm(tempDir, { recursive: true, force: true });
  });

  it('should read files from the source directory', async () => {
    // Create test files
    await fs.writeFile(path.join(testSourceDir, 'file1.ts'), 'content1');
    await fs.writeFile(path.join(testSourceDir, 'file2.ts'), 'content2');

    const files = await processFiles();

    expect(files).toEqual(['file1.ts', 'file2.ts']);
  });

  it('should return empty array when there are no source directories', async () => {
    // Mock getConfig to return empty SOURCE_DIRS
    (getConfig as jest.Mock).mockReturnValue({
      ...CONFIG,
      SOURCE_DIRS: [],
    });

    const files = await processFiles();

    expect(files).toEqual([]);
  });

  it('should propagate errors from fs.readdir', async () => {
    // Use a non-existent directory
    (getConfig as jest.Mock).mockReturnValue({
      ...CONFIG,
      SOURCE_DIRS: [path.join(tempDir, 'nonexistent')],
    });

    await expect(processFiles()).rejects.toThrow();
  });
});
