import fs from 'fs/promises';
import path from 'path';

const rootDir = new URL('../../..', import.meta.url).pathname;

// Files to preserve during removal
const PRESERVE_FILES = [
    'package.json',
    'pnpm-lock.yaml',
    'pyproject.toml'
];

// Files and directories to ignore during copying
const IGNORE_LIST = [
    // Test files and directories
    'test',
    'tests',
    '__tests__',
    '*.test.*',
    '*.spec.*',
    '*.test-d.*',

    // Python specific
    '__pycache__',
    '.pytest_cache',
    '*.pyc',

    // System files
    '.DS_Store',

    // Development directories
    'node_modules',
    '.git',
    '*.log',
    '*.tmp',
    '.env',
    '.venv',
    'venv',
    'dist',
    'build'
];

// Text replacements to perform on copied files
const REPLACEMENTS = [
    {
        from: '@hatchet',
        to: '@hatchet-dev/typescript-sdk'
    }
];

// Patterns to remove from code files
const REMOVAL_PATTERNS = [
    {
        regex: /^\s*(\/\/|#)\s*HH-.*$/gm,
        description: "HH- style comments"
    },
    {
        regex: /^\s*(\/\/|#)\s*!!\s*$/gm,
        description: "End marker comments"
    },
    {
        regex: /^\s*\/\*\s*eslint-.*\*\/$/gm,
        description: "ESLint disable block comments"
    },
    {
        regex: /\s*(\/\/|#)\s*eslint-disable-next-line.*$/gm,
        description: "ESLint disable line comments"
    }
];

function shouldIgnore(name) {
    return IGNORE_LIST.some(pattern => {
        if (pattern.includes('*')) {
            // Convert glob pattern to regex
            const regexPattern = pattern.replace(/\./g, '\\.').replace(/\*/g, '.*');
            return new RegExp(`^${regexPattern}$`).test(name);
        }
        return name === pattern;
    });
}

async function backupPreservedFiles(dir) {
    const preserved = {};
    try {
        const entries = await fs.readdir(dir, { withFileTypes: true });
        for (const entry of entries) {
            if (PRESERVE_FILES.includes(entry.name)) {
                const filePath = path.join(dir, entry.name);
                preserved[entry.name] = await fs.readFile(filePath, 'utf8');
                console.log(`Backed up: ${entry.name}`);
            }
        }
    } catch (error) {
        if (error.code !== 'ENOENT') {
            console.error(`Error backing up files in ${dir}:`, error);
        }
    }
    return preserved;
}

async function restorePreservedFiles(dir, preserved) {
    try {
        for (const [fileName, content] of Object.entries(preserved)) {
            const filePath = path.join(dir, fileName);
            await fs.writeFile(filePath, content, 'utf8');
            console.log(`Restored: ${fileName}`);
        }
    } catch (error) {
        console.error(`Error restoring files in ${dir}:`, error);
    }
}

async function removeDirectoryContents(dir) {
    try {
        const entries = await fs.readdir(dir, { withFileTypes: true });

        for (const entry of entries) {
            const fullPath = path.join(dir, entry.name);
            if (entry.isDirectory()) {
                await fs.rm(fullPath, { recursive: true, force: true });
            } else {
                await fs.unlink(fullPath);
            }
        }
        console.log(`Cleaned directory: ${dir}`);
    } catch (error) {
        if (error.code !== 'ENOENT') {
            console.error(`Error cleaning directory ${dir}:`, error);
        }
    }
}

function filterSpecialComments(content) {
    let filteredContent = content;

    for (const pattern of REMOVAL_PATTERNS) {
        filteredContent = filteredContent.replace(pattern.regex, '');
    }

    // Remove any resulting empty lines
    filteredContent = filteredContent.replace(/\n\s*\n/g, '\n\n');

    return filteredContent;
}

async function processFileContent(filePath) {
    try {
        let content = await fs.readFile(filePath, 'utf8');
        let hasChanges = false;

        // Apply text replacements
        for (const { from, to } of REPLACEMENTS) {
            const regex = new RegExp(from.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g');
            if (content.match(regex)) {
                content = content.replace(regex, to);
                hasChanges = true;
            }
        }

        // Filter out special comments
        const filteredContent = filterSpecialComments(content);
        if (filteredContent !== content) {
            content = filteredContent;
            hasChanges = true;
        }

        if (hasChanges) {
            await fs.writeFile(filePath, content, 'utf8');
        }
    } catch (error) {
        console.error(`Error processing file ${filePath}:`, error);
    }
}

async function copyDirectory(source, destination) {
    try {
        // Ensure destination directory exists
        await fs.mkdir(destination, { recursive: true });

        // Read source directory
        const entries = await fs.readdir(source, { withFileTypes: true });

        for (const entry of entries) {
            // Skip if the entry matches any ignore pattern
            if (shouldIgnore(entry.name)) {
                console.log(`Skipping ignored path: ${entry.name}`);
                continue;
            }

            const sourcePath = path.join(source, entry.name);
            const destPath = path.join(destination, entry.name);

            if (entry.isDirectory()) {
                // Recursively copy subdirectories
                await copyDirectory(sourcePath, destPath);
            } else {
                // Copy files
                await fs.copyFile(sourcePath, destPath);
                // Process the copied file for replacements and filtering
                await processFileContent(destPath);
            }
        }
    } catch (error) {
        console.error(`Error copying ${source} to ${destination}:`, error);
    }
}

async function main() {
    const mappings = [
        {
            source: path.join(rootDir, 'sdks/typescript/src/v1/examples'),
            dest: path.join(rootDir, 'examples/typescript')
        },
        {
            source: path.join(rootDir, 'sdks/python/examples'),
            dest: path.join(rootDir, 'examples/python')
        }
    ];

    // Create examples directory if it doesn't exist
    const examplesDir = path.join(rootDir, 'examples');
    await fs.mkdir(examplesDir, { recursive: true });

    // Remove and copy each directory
    for (const { source, dest } of mappings) {
        // Create destination if it doesn't exist
        await fs.mkdir(dest, { recursive: true });

        // Backup preserved files
        const preserved = await backupPreservedFiles(dest);

        console.log(`Cleaning ${dest}...`);
        await removeDirectoryContents(dest);

        console.log(`Copying ${source} to ${dest}...`);
        await copyDirectory(source, dest);

        // Restore preserved files
        await restorePreservedFiles(dest, preserved);
    }

    console.log('Examples copied successfully!');
}

main().catch(console.error);
