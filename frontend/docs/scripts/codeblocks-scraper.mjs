import { Octokit } from "@octokit/rest";
import { promises as fs } from "fs";
import dotenv from "dotenv";
dotenv.config();

const rawLinks = [
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/main/examples/simple/worker.py",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-delimeters/main/examples/simple/main.go",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-typescript-delimeters/main/src/examples/simple-worker.ts",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/main/examples/concurrency_limit/worker.py",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-delimeters/main/examples/limit-concurrency/group-round-robin/main.go",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-typescript-delimeters/main/src/examples/concurrency/group-round-robin/concurrency-worker.ts",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/main/examples/concurrency_limit/worker.py",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-delimeters/main/examples/limit-concurrency/cancel-in-progress/main.go",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-typescript-delimeters/main/src/examples/concurrency/cancel-in-progress/concurrency-worker.ts",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/main/examples/concurrency_limit_rr/worker.py",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-delimeters/main/examples/limit-concurrency/group-round-robin/main.go",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-typescript-delimeters/main/src/examples/concurrency/group-round-robin/concurrency-worker.ts",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/main/examples/simple/worker.py",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-delimeters/main/examples/simple/main.go",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-typescript-delimeters/main/src/examples/simple-worker.ts",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/main/examples/simple/worker.py",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-delimeters/main/examples/simple/main.go",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-typescript-delimeters/main/src/examples/simple-worker.ts",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/main/examples/timeout/worker.py",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-delimeters/main/examples/timeout/main.go",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/main/examples/timeout/worker.py",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/main/examples/timeout/worker.py",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-delimeters/main/examples/timeout/main.go",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/main/examples/on_failure/worker.py",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-delimeters/main/examples/on-failure/main.go",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-typescript-delimeters/main/src/examples/on-failure.ts",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/main/examples/manual_trigger/stream.py",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-typescript-delimeters/main/src/examples/manual-trigger.ts",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/main/examples/fanout/stream.py",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-delimeters/main/examples/stream-event-by-meta/main.go",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-typescript-delimeters/main/src/examples/stream-by-additional-meta.ts",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/main/examples/rate_limit/worker.py",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-delimeters/main/examples/rate-limit/main.go",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-typescript-delimeters/main/src/examples/rate-limit/worker.ts",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-delimeters/main/examples/rate-limit/main.go",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-typescript-delimeters/main/src/examples/rate-limit/worker.ts",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/main/examples/sticky-workers/worker.py",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-delimeters/main/examples/assignment-sticky/run.go",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-typescript-delimeters/main/src/examples/sticky-worker.ts",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/main/examples/sticky-workers/worker.py",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-delimeters/main/examples/assignment-sticky/run.go",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-typescript-delimeters/main/src/examples/sticky-worker.ts",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/main/examples/affinity-workers/worker.py",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-delimeters/main/examples/assignment-affinity/run.go",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-typescript-delimeters/main/src/examples/affinity-workers.ts",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/main/examples/affinity-workers/worker.py",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-delimeters/main/examples/assignment-affinity/run.go",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-typescript-delimeters/main/src/examples/affinity-workers.ts",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/main/examples/affinity-workers/worker.py",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-delimeters/main/examples/assignment-affinity/run.go",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-typescript-delimeters/main/src/examples/affinity-workers.ts",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-python-delimeters/main/examples/cancellation/worker.py",
  "https://raw.githubusercontent.com/TranquilVarun/hatchet-delimeters/main/examples/cancellation/run.go",
];

const links = [...new Set(rawLinks)];

const commentSyntax = {
  Python: "#",
  Typescript: "//",
  Golang: "//",
};

const octokit = new Octokit({
  auth: process.env.GITHUB_TOKEN,
});

async function scrapeGitHubCode() {
  const result = {};
  for (const link of links) {
    try {
      const urlParts = link.split("/");
      const owner = urlParts[3];
      const repo = urlParts[4];
      const branch = urlParts[5];
      const path = urlParts.slice(6).join("/");

      const response = await octokit.repos.getContent({
        owner,
        repo,
        path,
        ref: branch,
      });

      const content = Buffer.from(response.data.content, "base64").toString();

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

        let blockMatch;
        let matchFound = false;
        while ((blockMatch = blockRegex.exec(content)) !== null) {
          matchFound = true;
          const [, blockName, code] = blockMatch;
          const codeLines = code.split("\n");
          const minIndent = Math.min(
            ...codeLines
              .filter((line) => line.trim().length > 0)
              .map((line) => line.match(/^\s*/)[0].length)
          );
          const trimmedCode = codeLines
            .map((line) => line.slice(minIndent))
            .join("\n")
            .trim();

          result[language].push({
            blockName: blockName.trim(),
            code: trimmedCode,
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
  if (scrapedData.Golang) {
    scrapedData.Go = scrapedData.Golang;
    delete scrapedData.Golang;
  }
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
