const fs = require('fs');
const path = require('path');

const sourceDir = path.join(__dirname, '../../out/snips');
const targets = {
  app: {
    dir: path.join(__dirname, '../../../app/src/next/lib/docs/generated/snips'),
    replace: {
      from: /@\/lib\/generated\/snips\//g,
      to: '@/next/lib/docs/generated/snips/',
    },
  },
  docs: {
    dir: path.join(__dirname, '../../../docs/lib/generated/snips'),
    replace: null, // No path replacement needed for docs
  },
};

function replaceInFile(filePath, replaceConfig) {
  if (!replaceConfig) return;

  const content = fs.readFileSync(filePath, 'utf8');
  const newContent = content.replace(replaceConfig.from, replaceConfig.to);
  fs.writeFileSync(filePath, newContent);
}

function copyDirectory(src, dest, replaceConfig) {
  // Create destination directory if it doesn't exist
  if (!fs.existsSync(dest)) {
    fs.mkdirSync(dest, { recursive: true });
  }

  const entries = fs.readdirSync(src, { withFileTypes: true });

  for (const entry of entries) {
    const srcPath = path.join(src, entry.name);
    const destPath = path.join(dest, entry.name);

    if (entry.isDirectory()) {
      copyDirectory(srcPath, destPath, replaceConfig);
    } else {
      fs.copyFileSync(srcPath, destPath);
      replaceInFile(destPath, replaceConfig);
    }
  }
}

// Process all targets
for (const [name, config] of Object.entries(targets)) {
  console.log(`Processing ${name}...`);

  // Remove target directory if it exists
  if (fs.existsSync(config.dir)) {
    fs.rmSync(config.dir, { recursive: true, force: true });
  }

  // Copy and process files
  copyDirectory(sourceDir, config.dir, config.replace);
  console.log(`Completed ${name}`);
}
