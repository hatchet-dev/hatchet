const commentPrefixes = ["//", "#"];

const isCommentLine = (line: string) => {
  const trimmed = line.trim();
  return commentPrefixes.some((prefix) => trimmed.startsWith(prefix));
};

const startsWithPrefixAndChar = (line: string, specialChar: string) => {
  const trimmed = line.trim();
  return commentPrefixes.some((prefix) =>
    trimmed.startsWith(`${prefix} ${specialChar}`)
  );
};

const isTargetLine = (line: string, target: string) => {
  const split = line.split(/[>?]/);
  return split[1].trim() === target.trim();
};

export const parseDocComments = (
  source: string,
  target?: string,
  collapsed: boolean = false
): string => {
  if (!target) {
    return source;
  }

  const lines = source.split("\n");
  let isSnippet = false;
  let isCollecting = false;
  const resultLines: string[] = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    // Start collecting at > or ?
    if (
      isCommentLine(line) &&
      (startsWithPrefixAndChar(line, ">") ||
        startsWithPrefixAndChar(line, "?")) &&
      isTargetLine(line, target)
    ) {
      isSnippet = true;
      isCollecting = true;
      continue;
    }

    if (isSnippet && isCommentLine(line)) {
      // Start collecting again at 👀 or ,
      if (
        startsWithPrefixAndChar(line, "👀") ||
        startsWithPrefixAndChar(line, ",")
      ) {
        isCollecting = true;
        continue;
      }

      // Stop at '...' depending on collapsed flag
      if (isCollecting && startsWithPrefixAndChar(line, "...")) {
        if (collapsed) {
          isCollecting = false;
          resultLines.push(line);
        }
        continue;
      }

      // Stop at !! or !!
      if (
        startsWithPrefixAndChar(line, "!!") ||
        startsWithPrefixAndChar(line, "!!")
      ) {
        break;
      }
    }

    // Collect focused section
    if (isCollecting) {
      resultLines.push(line);
    }
  }

  if (resultLines.length === 0) {
    return `🚨 No snippet found for ${target} \n\n${source}`;
  }

  return resultLines.join("\n");
};
