import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Get the project root directory (3 levels up from scripts dir)
const projectRoot = path.resolve(__dirname, '../../..');

function cleanQuestionText(text) {
  return text.trim().replace(/❓/g, '').trim();
}

function normalizeKey(key) {
  return key.trim().toLowerCase().replace(/[\s-]+/g, '_').replace(/[()]/g, '');
}

function extractQuestions(filePath) {
  const content = fs.readFileSync(filePath, 'utf8');
  const lines = content.split('\n');
  const questions = [];

  // Match both Python (#) and JS/TS (//) style comments followed by ❓ or ?
  const questionRegex = /^[\s]*(\/\/|#).*[❓?]/;

  lines.forEach((line) => {
    if (questionRegex.test(line)) {
      const comment = line.trim().replace(/^[\s]*(\/\/|#)[\s]*/, '');
      questions.push(cleanQuestionText(comment));
    }
  });

  return questions;
}

// Extract highlights in the format // HH-key lineCount strings,comma,separated
function extractHighlights(filePath, content) {
  const lines = content.split('\n');
  const highlights = {};

  // Match both Python (#) and JS/TS (//) style comments with HH- prefix
  const highlightRegex = /^[\s]*(\/\/|#)\s*HH-(\w+)\s+(\d+)(?:\s+(.+))?$/;

  // Track which lines will be removed in the cleaned version
  const removedLines = [];
  for (let i = 0; i < lines.length; i++) {
    // Mark HH- lines as removed
    if (/^[\s]*(\/\/|#)\s*HH-/.test(lines[i])) {
      removedLines.push(i);
    }
    // Mark !! lines as removed
    if (/^[\s]*(\/\/|#)?\s*!!/.test(lines[i])) {
      removedLines.push(i);
    }
  }

  // For debugging
  console.log(`Removed lines for ${path.basename(filePath)}:`, removedLines);

  for (let i = 0; i < lines.length; i++) {
    const match = lines[i].match(highlightRegex);
    if (match) {
      const key = match[2];
      const lineCount = parseInt(match[3], 10);
      const strings = match[4] ? match[4].split(',').map(s => s.trim()) : [];

      // Mark this highlight line as removed
      if (!removedLines.includes(i)) {
        removedLines.push(i);
      }

      // Find the next non-comment line
      let startLine = i + 1;
      while (startLine < lines.length &&
             (lines[startLine].trim().startsWith('//') ||
              lines[startLine].trim().startsWith('#') ||
              lines[startLine].trim() === '')) {
        // If these are also lines that will be removed, add them
        if (!removedLines.includes(startLine) &&
            (lines[startLine].includes('HH-') ||
             lines[startLine].includes('!!'))) {
          removedLines.push(startLine);
        }
        startLine++;
      }

      console.log(`For ${key}, startLine = ${startLine}`);

      // Calculate the line numbers to highlight, adjusting for removed lines
      const lineNumbers = [];
      for (let j = 0; j < lineCount && (startLine + j) < lines.length; j++) {
        const originalLineNumber = startLine + j;

        // Count how many removed lines appear before this line
        const removedLinesBefore = removedLines.filter(line => line < originalLineNumber).length;

        // Convert from 0-based to 1-based line number and adjust for removed lines
        const adjustedLineNumber = originalLineNumber - removedLinesBefore + 1;

        console.log(`  Line ${j+1}: originalLine=${originalLineNumber}, removedBefore=${removedLinesBefore}, adjusted=${adjustedLineNumber}`);

        lineNumbers.push(adjustedLineNumber);
      }

      highlights[key] = {
        lines: lineNumbers,
        strings: strings
      };
    }
  }

  return highlights;
}

// Get SDK source file path from examples file path
function getSDKSourcePath(filePath) {
  // Replace examples/typescript with sdks/typescript/src/v1/examples
  if (filePath.includes('/examples/typescript/')) {
    return filePath.replace('/examples/typescript/', '/sdks/typescript/src/v1/examples/');
  }
  // Other language mappings can be added here if needed
  return filePath;
}

// Store all snippets in a single map
const snippetsMap = new Map();

function createSnippetId(filePath) {
  // Generate a unique ID based on the path
  return Buffer.from(filePath).toString('base64').replace(/[/+=]/g, '_');
}

function buildSnipsObject(dir) {
  const snips = {};
  const languages = ['typescript', 'python', 'go'];

  // Check if directory has language subdirectories
  const hasLanguageSubdirs = languages.some(language => fs.existsSync(path.join(dir, language)));

  if (hasLanguageSubdirs) {
    // Process as before with language subdirectories
    languages.forEach(language => {
      const languagePath = path.join(dir, language);
      if (fs.existsSync(languagePath)) {
        snips[language] = processDirectory(languagePath);
      }
    });
  } else {
    // For SDK examples, process directly
    snips.typescript = processDirectory(dir);
  }

  return snips;
}

function processDirectory(dirPath) {
  const result = {};
  const items = fs.readdirSync(dirPath);

  items.forEach(item => {
    const fullPath = path.join(dirPath, item);
    const stats = fs.statSync(fullPath);

    if (stats.isDirectory()) {
      // Process subdirectory
      const subDirContent = processDirectory(fullPath);
      if (Object.keys(subDirContent).length > 0) {
        result[normalizeKey(item)] = subDirContent;
      }
    } else {
      // Process file
      const fileName = normalizeKey(path.basename(item, path.extname(item)));
      const ext = path.extname(fullPath).slice(1) || 'txt';

      // Check if this is a root file (directly in language directory)
      const isRootFile = path.dirname(fullPath) === dirPath &&
        ['ts', 'js', 'py', 'go'].includes(ext);

      // Include if it's a root file or one of our specific file types
      if (isRootFile || ['run', 'worker', 'workflow'].includes(fileName)) {
        // Read cleaned content for the snippet
        const cleanedContent = fs.readFileSync(fullPath, 'utf8');
        const snippetId = createSnippetId(fullPath);

        // For highlight extraction, use the original SDK source if available
        let highlights = {};
        const sdkSourcePath = getSDKSourcePath(fullPath);
        if (fs.existsSync(sdkSourcePath) && sdkSourcePath !== fullPath) {
          // This is a copied example, try to get highlights from original source
          try {
            const originalContent = fs.readFileSync(sdkSourcePath, 'utf8');
            highlights = extractHighlights(sdkSourcePath, originalContent);
          } catch (err) {
            console.warn(`Failed to read original source for highlights: ${sdkSourcePath}`);
          }
        }

        // Store the content and metadata
        snippetsMap.set(snippetId, {
          content: cleanedContent,
          language: ext,
          source: path.relative(projectRoot, fullPath),
          highlights
        });

        const questions = extractQuestions(fullPath);

        // Create an object with "*" as the first key and empty question
        const fileObj = {
          "*": ":"+snippetId
        };

        // Add each question as a key pointing to the same snippetId
        questions.forEach(question => {
          const normalizedQuestion = normalizeKey(question);
          fileObj[normalizedQuestion] = `${question}:${snippetId}`;
        });

        // For root files, store under a special key
        if (isRootFile) {
          result[fileName] = fileObj;
        } else {
          result[fileName] = fileObj;
        }
      }
    }
  });

  return result;
}

// Path to examples directory (relative to project root)
const examplesDir = path.join(projectRoot, 'examples');

console.log(`Processing examples from: ${examplesDir}`);

// Generate the snips object and snippets content
const examplesSnips = buildSnipsObject(examplesDir);
console.log(`Found ${Object.keys(examplesSnips).length} language examples`);

// Convert snippetsMap to a regular object for export
const snippetsContent = Object.fromEntries(snippetsMap);

// Write the combined output file
const outputPath = path.join(__dirname, '..', 'lib/snips.ts');

// Ensure the lib directory exists
const libDir = path.dirname(outputPath);
if (!fs.existsSync(libDir)) {
  fs.mkdirSync(libDir, { recursive: true });
}

const fileContent = `// This file is auto-generated. Do not edit directly.

// Types for snippets
export type Snippet = {
  content: string;
  language: string;
  source: string;
  highlights?: {
    [key: string]: {
      lines: number[];
      strings: string[];
    }
  };
};

type Snippets = {
  [key: string]: Snippet;
};

// Snippet contents
export const snippets: Snippets = ${JSON.stringify(snippetsContent, null, 2)} as const;

// Snippet mapping
const snips = ${JSON.stringify(examplesSnips, null, 2)} as const;

export default snips;
`;

fs.writeFileSync(outputPath, fileContent, 'utf8');
console.log(`Successfully generated snips file at: ${outputPath}`);
