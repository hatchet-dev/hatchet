import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Function to recursively find all MDX files
function findMdxFiles(dir) {
    const files = fs.readdirSync(dir);
    let mdxFiles = [];

    for (const file of files) {
        const filePath = path.join(dir, file);
        const stat = fs.statSync(filePath);

        if (stat.isDirectory()) {
            mdxFiles = mdxFiles.concat(findMdxFiles(filePath));
        } else if (file.endsWith('.mdx')) {
            mdxFiles.push(filePath);
        }
    }

    return mdxFiles;
}

// Function to process a single MDX file
function processMdxFile(filePath) {
    let content = fs.readFileSync(filePath, 'utf8');
    let modified = false;

    // Regular expression to match Snippet components
    // This will match patterns like <Snippet src={snips.typescript.simple.worker} />
    const snippetRegex = /<Snippet\s+src={([^}]+)}\s*\/>/g;

    // Replace matches with the new format
    const newContent = content.replace(snippetRegex, (match, src) => {
        // Extract the last part of the path for the block attribute
        const pathParts = src.split('.');
        const blockName = pathParts[pathParts.length - 1];
        // Remove the last part from the src
        const newSrc = pathParts.slice(0, -1).join('.');
        
        modified = true;
        return `<Snippet src={${newSrc}} block="${blockName}" />`;
    });

    if (modified) {
        fs.writeFileSync(filePath, newContent, 'utf8');
        console.log(`Updated ${filePath}`);
    }
}

// Main function
function main() {
    const docsDir = path.join(__dirname, 'pages');
    const mdxFiles = findMdxFiles(docsDir);

    console.log(`Found ${mdxFiles.length} MDX files to process`);

    for (const file of mdxFiles) {
        processMdxFile(file);
    }

    console.log('Migration complete!');
}

main(); 