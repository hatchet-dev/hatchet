const commentPrefixes = ['//', '#'];

const isCommentLine = (line: string) => {
  const trimmed = line.trim();
  return commentPrefixes.some(prefix => trimmed.startsWith(prefix));
}

const startsWithPrefixAndChar = (line: string, specialChar: string) => {
  const trimmed = line.trim();
  return commentPrefixes.some(prefix => trimmed.startsWith(`${prefix} ${specialChar}`));
}

const isTargetLine = (line: string, target: string) => {
  const split = line.split(/[❓?]/);
  return split[1].trim() === target.trim();
}


export const parseDocComments = (
    source: string,
    target: string,
    collapsed: boolean = false,
): string => {
    const lines = source.split('\n');
    let isSnippet = false;
    let isCollecting = false;
    const resultLines: string[] = [];

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];

      // Start collecting at ❓ or ?
      if (
        isCommentLine(line) &&
        (startsWithPrefixAndChar(line, '❓') || startsWithPrefixAndChar(line, '?')) &&
        isTargetLine(line, target)
      ) {
          isSnippet = true;
          isCollecting = true;
          continue;
      }

      if (isSnippet && isCommentLine(line)) {
        // Start collecting again at 👀
        if (startsWithPrefixAndChar(line, '👀')) {
            isCollecting = true;
        }

        // Start collecting again at ,
        if (startsWithPrefixAndChar(line, ',')) {
            isCollecting = true;
            continue;
        }

        // Collect until '...'
        if (isCollecting && startsWithPrefixAndChar(line, '...')) {
            if (collapsed) {
                isCollecting = false;
                resultLines.push(line);
            } else {
                continue;
            }
        }

        // Stop at ‼️ or !!
        if (isSnippet && (startsWithPrefixAndChar(line, '‼️') || startsWithPrefixAndChar(line, '!!'))) {
            break;
        }
      }

      // Collect focused section
      if (isCollecting) {
        if (line.trim() === '') {
          resultLines.push(' ');
        } else {
          resultLines.push(line);
        }
      }
    }

    if (resultLines.length === 0) {
      return `🚨 No snippet found for ${target} \n\n${source}`;
    }

    // Shift indentation to the least indented level
    const nonEmptyLines = resultLines.filter(line => line.trim() !== '');
    const indents = nonEmptyLines.map(line => line.match(/^(\s*)/)?.[1].length || 0);
    const minIndent = Math.min(...indents);

    const shiftedLines = resultLines.map(line => {
      return line.startsWith(' '.repeat(minIndent))
        ? line.slice(minIndent)
        : line.replace(/^\s+/, '');
    });

    return shiftedLines.join('\n');
};
