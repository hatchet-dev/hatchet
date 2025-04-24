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

// Store all snippets in a single map
const snippetsMap = new Map();

function createSnippetId(filePath) {
  // Generate a unique ID based on the path
  return Buffer.from(filePath).toString('base64').replace(/[/+=]/g, '_');
}

function buildSnipsObject(dir) {
  const snips = {};
  const languages = ['typescript', 'python', 'go'];

  languages.forEach(language => {
    const languagePath = path.join(dir, language);
    if (fs.existsSync(languagePath)) {
      snips[language] = processDirectory(languagePath);
    }
  });

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
        result[item] = subDirContent;
      }
    } else {
      // Process file
      const fileName = path.basename(item, path.extname(item));
      // Only include specific file types we're interested in
      if (['run', 'worker', 'workflow'].includes(fileName)) {
        const content = fs.readFileSync(fullPath, 'utf8');
        const snippetId = createSnippetId(fullPath);
        const ext = path.extname(fullPath).slice(1) || 'txt';
        
        // Store the snippet content and metadata
        snippetsMap.set(snippetId, {
          content,
          language: ext,
          source: fullPath
        });
        
        const questions = extractQuestions(fullPath);
        
        // Create an object with "*" as the first key and empty question
        const fileObj = {
          "*": ":"+snippetId
        };
        
        // Add each question as a key pointing to the same snippetId
        questions.forEach(question => {
          fileObj[question] = `${question}:${snippetId}`;
        });
        
        result[fileName] = fileObj;
      }
    }
  });

  return result;
}

// Path to examples directory (relative to project root)
const examplesDir = path.join(projectRoot, 'examples');

// Generate the snips object and snippets content
const snips = buildSnipsObject(examplesDir);

// Convert snippetsMap to a regular object for export
const snippetsContent = Object.fromEntries(snippetsMap);

// Write the combined output file
const outputPath = path.join(__dirname, '../lib/snips.ts');
const fileContent = `// This file is auto-generated. Do not edit directly.

// Types for snippets
type Snippet = {
  content: string;
  language: string;
  source: string;
};

type Snippets = {
  [key: string]: Snippet;
};

// Snippet contents
export const snippets: Snippets = ${JSON.stringify(snippetsContent, null, 2)} as const;

// Snippet mapping
const snips = ${JSON.stringify(snips, null, 2)} as const;

export default snips;
`;

fs.writeFileSync(outputPath, fileContent, 'utf8');
console.log('Successfully generated snips file at:', outputPath); 