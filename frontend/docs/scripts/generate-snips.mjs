import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Get the project root directory (3 levels up from scripts dir)
const projectRoot = path.resolve(__dirname, '../../..');

// Create snippets directory if it doesn't exist
const snippetsDir = path.join(__dirname, '../lib/snippets');
if (!fs.existsSync(snippetsDir)) {
  fs.mkdirSync(snippetsDir, { recursive: true });
}

function cleanQuestionText(text) {
  return text.trim().replace(/❓/g, '').trim();
}

function extractQuestions(filePath) {
  const content = fs.readFileSync(filePath, 'utf8');
  const lines = content.split('\n');
  const questions = [];
  
  // Match both Python (#) and JS/TS (//) style comments followed by ❓ or ?
  const questionRegex = /^[\s]*(\/\/|#).*[❓?]/;
  
  lines.forEach((line, index) => {
    if (questionRegex.test(line)) {
      const comment = line.trim().replace(/^[\s]*(\/\/|#)[\s]*/, '');
      questions.push(cleanQuestionText(comment));
    }
  });
  
  return questions;
}

function createSnippetModule(filePath) {
  // Generate a unique filename based on the path
  const hash = Buffer.from(filePath).toString('base64').replace(/[/+=]/g, '_');
  const content = fs.readFileSync(filePath, 'utf8');
  const ext = path.extname(filePath);
  const snippetPath = path.join(snippetsDir, `${hash}.ts`);
  
  // Create a TypeScript module that exports the content
  const moduleContent = `// Generated from ${filePath}
export const content = ${JSON.stringify(content)};
export const language = ${JSON.stringify(ext.slice(1) || 'txt')};
`;
  
  // Write the module
  fs.writeFileSync(snippetPath, moduleContent, 'utf8');
  
  // Return the module path relative to lib directory
  return `snippets/${path.basename(snippetPath)}`;
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
        // Create a module for this snippet and get its path
        const modulePath = createSnippetModule(fullPath);
        const questions = extractQuestions(fullPath);
        
        // Create an object with "*" as the first key
        const fileObj = {
          "*": ":"+modulePath
        };
        
        // Add each question as a key pointing to the same module
        questions.forEach(question => {
          fileObj[question] = question+":"+modulePath;
        });
        
        result[fileName] = fileObj;
      }
    }
  });

  return result;
}

// Path to examples directory (relative to project root)
const examplesDir = path.join(projectRoot, 'examples');

// Generate the snips object
const snips = buildSnipsObject(examplesDir);

// Write the snips object to a file
const outputPath = path.join(__dirname, '../lib/snips.ts');
const fileContent = `// This file is auto-generated. Do not edit directly.
const snips = ${JSON.stringify(snips, null, 2)} as const;

export default snips;
`;

fs.writeFileSync(outputPath, fileContent, 'utf8');
console.log('Successfully generated snips file at:', outputPath); 