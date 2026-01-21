// Terminal themes for ghostty-web
// Dark and light mode color palettes

export interface TerminalTheme {
  background: string;
  foreground: string;
  cursor: string;
  cursorAccent: string;
  selectionBackground: string;
  selectionForeground: string;
  black: string;
  red: string;
  green: string;
  yellow: string;
  blue: string;
  magenta: string;
  cyan: string;
  white: string;
  brightBlack: string;
  brightRed: string;
  brightGreen: string;
  brightYellow: string;
  brightBlue: string;
  brightMagenta: string;
  brightCyan: string;
  brightWhite: string;
}

export const darkTheme: TerminalTheme = {
  background: '#1e293b',
  foreground: '#dddddd',
  cursor: '#007acc',
  cursorAccent: '#bfdbfe',
  selectionBackground: '#bfdbfe',
  selectionForeground: '#000000',
  // ANSI colors (palette 0-15)
  black: '#191919',
  red: '#aa342e',
  green: '#4b8c0f',
  yellow: '#dbba00',
  blue: '#1370d3',
  magenta: '#c43ac3',
  cyan: '#008eb0',
  white: '#bebebe',
  brightBlack: '#525252',
  brightRed: '#f05b50',
  brightGreen: '#95dc55',
  brightYellow: '#ffe763',
  brightBlue: '#60a4ec',
  brightMagenta: '#e26be2',
  brightCyan: '#60b6cb',
  brightWhite: '#f7f7f7',
};

export const lightTheme: TerminalTheme = {
  background: '#f9f9f9',
  foreground: '#373a41',
  cursor: '#f32759',
  cursorAccent: '#ffffff',
  selectionBackground: '#daf0ff',
  selectionForeground: '#373a41',
  // ANSI colors (palette 0-15)
  black: '#373a41',
  red: '#d52753',
  green: '#23974a',
  yellow: '#df631c',
  blue: '#275fe4',
  magenta: '#823ff1',
  cyan: '#27618d',
  white: '#babbc2',
  brightBlack: '#676a77',
  brightRed: '#ff6480',
  brightGreen: '#3cbc66',
  brightYellow: '#c5a332',
  brightBlue: '#0099e1',
  brightMagenta: '#ce33c0',
  brightCyan: '#6d93bb',
  brightWhite: '#d3d3d3',
};
