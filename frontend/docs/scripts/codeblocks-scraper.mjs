import { Octokit } from "@octokit/rest";
import { promises as fs } from "fs";

const links = [
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/delimeters/examples/concurrency_limit/worker.py",
];

const commentSyntax = {
  Python: "#",
  Typescript: "//",
  Go: "//",
};

const octokit = new Octokit({
  auth: process.env.GITHUB_PAT,
});

async function scrapeGitHubCode() {
  const result = {};
  for (const link of links) {
    try {
      const urlParts = link.split('/');
      console.log("urlParts:", urlParts);
      const owner = urlParts[3];
      const repo = urlParts[4];
      const branch = urlParts[5];
      const path = urlParts.slice(6).join('/');
      console.log("owner:", owner, "repo:", repo, "branch:", branch, "path:", path);
      
      const response = await octokit.repos.getContent({
        owner,
        repo,
        path,
        ref: branch,
      });

      const content = Buffer.from(response.data.content, 'base64').toString();
      console.log("Content:", content);
      
      const lines = content.split("\n");
      const languageLine = lines[0].trim();
      const language = Object.keys(commentSyntax).find(
        (lang) =>
          languageLine.replace(/\s+/g, "") === `${commentSyntax[lang]}${lang}`
      );
      
      if (language) {
        if (!result[language]) {
          result[language] = [];
        }
        const comment = commentSyntax[language];
        const blockRegex = new RegExp(
          `${comment}START\\s+(.+?)\\s*\\n([\\s\\S]*?)\\n\\s*${comment}END`,
          "g"
        );
        console.log("blockRegex:", blockRegex);
        
        let blockMatch;
        let matchFound = false;
        while ((blockMatch = blockRegex.exec(content)) !== null) {
          console.log("blockMatch:", blockMatch);
          matchFound = true;
          const [, blockName, code] = blockMatch;
          result[language].push({
            blockName: blockName.trim(),
            code: code.trim(),
            source: link,
          });
        }
        
        if (!matchFound) {
          console.log("No blocks found in the content.");
        }
      } else {
        console.error(`Language not recognized in file: ${link}`);
        console.log("First line:", languageLine);
      }
    } catch (error) {
      console.error(
        `Error scraping ${link}:`,
        error instanceof Error ? error.message : String(error)
      );
    }
  }
  return result;
}

async function main() {
  const scrapedData = await scrapeGitHubCode();
  console.log("Scraped data:", JSON.stringify(scrapedData, null, 2));
  await fs.writeFile("codeblocks.json", JSON.stringify(scrapedData, null, 2));
  console.log("Code blocks have been scraped and saved to codeblocks.json");
}

main().catch((error) => {
  console.error(
    "An error occurred:",
    error instanceof Error ? error.message : String(error)
  );
  process.exit(1);
});