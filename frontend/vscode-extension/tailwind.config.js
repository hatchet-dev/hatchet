/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: 'class',
  content: [
    './src/webview/**/*.{ts,tsx}',
    // Scan the shared dag-visualizer package for class names
    '../packages/dag-visualizer/src/**/*.{ts,tsx}',
  ],
  theme: {
    extend: {
      // Mirror the Hatchet app's custom colors so shared components look identical
      colors: {
        border: 'hsl(var(--border))',
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))',
        },
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))',
        },
      },
      fontFamily: {
        mono: [
          'ui-monospace',
          'SFMono-Regular',
          'Menlo',
          'Monaco',
          'Consolas',
          '"Liberation Mono"',
          '"Courier New"',
          'monospace',
        ],
      },
      keyframes: {
        flip: { to: { transform: 'rotate(360deg)' } },
        rotate: { to: { transform: 'rotate(90deg)' } },
      },
      animation: {
        flip: 'flip 6s infinite steps(2, end)',
        rotate: 'rotate 3s linear infinite both',
      },
    },
  },
  plugins: [require('tailwindcss-animate')],
};
