import { CSSProperties } from 'react';

// ---- ANSI parser (adapted from react-logviewer/src/components/Utils/ansiparse.ts) ----

interface AnsiPart {
  text: string;
  foreground?: string;
  background?: string;
  bold?: boolean;
  italic?: boolean;
  underline?: boolean;
}

type AnsiState = Omit<AnsiPart, 'text'>;

const FOREGROUND_COLORS: Record<string, string> = {
  '30': 'black',
  '31': 'red',
  '32': 'green',
  '33': 'yellow',
  '34': 'blue',
  '35': 'magenta',
  '36': 'cyan',
  '37': 'white',
  '90': 'grey',
};

const BACKGROUND_COLORS: Record<string, string> = {
  '40': 'black',
  '41': 'red',
  '42': 'green',
  '43': 'yellow',
  '44': 'blue',
  '45': 'magenta',
  '46': 'cyan',
  '47': 'white',
};

function ansiparse(str: string): AnsiPart[] {
  let matchingControl: string | null = null;
  let matchingData: string | null = null;
  let matchingText = '';
  let ansiState: string[] = [];
  const result: AnsiPart[] = [];
  let state: AnsiState = {};

  for (let i = 0; i < str.length; i++) {
    if (matchingControl !== null) {
      if (matchingControl === '\x1b' && str[i] === '[') {
        if (matchingText) {
          result.push({ text: matchingText, ...state });
          state = {};
          matchingText = '';
        }
        matchingControl = null;
        matchingData = '';
      } else {
        matchingText += matchingControl + str[i];
        matchingControl = null;
      }
      continue;
    }

    if (matchingData !== null) {
      if (str[i] === ';') {
        ansiState.push(matchingData);
        matchingData = '';
      } else if (str[i] === 'm') {
        ansiState.push(matchingData);
        matchingData = null;
        matchingText = '';

        for (const code of ansiState) {
          if (FOREGROUND_COLORS[code]) {
            state.foreground = FOREGROUND_COLORS[code];
          } else if (BACKGROUND_COLORS[code]) {
            state.background = BACKGROUND_COLORS[code];
          } else if (code === '39') {
            delete state.foreground;
          } else if (code === '49') {
            delete state.background;
          } else if (code === '1') {
            state.bold = true;
          } else if (code === '3') {
            state.italic = true;
          } else if (code === '4') {
            state.underline = true;
          } else if (code === '22') {
            state.bold = false;
          } else if (code === '23') {
            state.italic = false;
          } else if (code === '24') {
            state.underline = false;
          }
        }
        ansiState = [];
      } else {
        matchingData += str[i];
      }
      continue;
    }

    if (str[i] === '\x1b') {
      matchingControl = str[i];
    } else if (str[i] === '\u0008') {
      // Backspace: erase previous character
      if (matchingText.length) {
        matchingText = matchingText.slice(0, -1);
      } else if (result.length) {
        const last = result[result.length - 1];
        if (last.text.length === 1) {
          result.pop();
        } else {
          last.text = last.text.slice(0, -1);
        }
      }
    } else {
      matchingText += str[i];
    }
  }

  if (matchingText) {
    result.push({ text: matchingText + (matchingControl ?? ''), ...state });
  }

  return result;
}

// ---- Color mapping to existing CSS variables ----

const FOREGROUND_CSS_VARS: Record<string, string> = {
  red: 'var(--terminal-red)',
  green: 'var(--terminal-green)',
  yellow: 'var(--terminal-yellow)',
  blue: 'var(--terminal-blue)',
  magenta: 'var(--terminal-magenta)',
  cyan: 'var(--terminal-cyan)',
};

function partStyle(part: AnsiPart): CSSProperties | undefined {
  const style: CSSProperties = {};
  let hasStyle = false;

  const fg = part.foreground && FOREGROUND_CSS_VARS[part.foreground];
  if (fg) {
    style.color = fg;
    hasStyle = true;
  }
  if (part.bold) {
    style.fontWeight = 'bold';
    hasStyle = true;
  }
  if (part.italic) {
    style.fontStyle = 'italic';
    hasStyle = true;
  }
  if (part.underline) {
    style.textDecoration = 'underline';
    hasStyle = true;
  }

  return hasStyle ? style : undefined;
}

// ---- Component ----

export function AnsiLine({ text }: { text: string }) {
  const parts = ansiparse(text);

  if (parts.length === 0) {
    return null;
  }

  return (
    <>
      {parts.map((part, i) => {
        const style = partStyle(part);
        return style ? (
          <span key={i} style={style}>
            {part.text}
          </span>
        ) : (
          part.text
        );
      })}
    </>
  );
}
