// Generate agent-prism Tailwind color mappings from CSS custom properties
const agentPrismTokens = [
  "background", "foreground", "primary", "primary-foreground", "secondary",
  "secondary-foreground", "muted", "muted-foreground", "accent", "accent-foreground",
  "brand", "brand-foreground", "brand-secondary", "brand-secondary-foreground",
  "border", "border-subtle", "border-strong", "border-inverse",
  "success", "success-muted", "success-muted-foreground",
  "error", "error-muted", "error-muted-foreground",
  "warning", "warning-muted", "warning-muted-foreground",
  "pending", "pending-muted", "pending-muted-foreground",
  "code-string", "code-number", "code-key", "code-base",
  "badge-default", "badge-default-foreground",
  "avatar-llm", "badge-llm", "badge-llm-foreground", "timeline-llm",
  "avatar-agent", "badge-agent", "badge-agent-foreground", "timeline-agent",
  "avatar-tool", "badge-tool", "badge-tool-foreground", "timeline-tool",
  "avatar-chain", "badge-chain", "badge-chain-foreground", "timeline-chain",
  "avatar-retrieval", "badge-retrieval", "badge-retrieval-foreground", "timeline-retrieval",
  "avatar-embedding", "badge-embedding", "badge-embedding-foreground", "timeline-embedding",
  "avatar-guardrail", "badge-guardrail", "badge-guardrail-foreground", "timeline-guardrail",
  "avatar-create-agent", "badge-create-agent", "badge-create-agent-foreground", "timeline-create-agent",
  "avatar-span", "badge-span", "badge-span-foreground", "timeline-span",
  "avatar-event", "badge-event", "badge-event-foreground", "timeline-event",
  "avatar-unknown", "badge-unknown", "badge-unknown-foreground", "timeline-unknown",
];
const agentPrismColors = Object.fromEntries(
  agentPrismTokens.map((name) => [
    `agentprism-${name}`,
    `hsl(var(--agentprism-${name}) / <alpha-value>)`,
  ])
);

/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: ["class"],
  content: [
    './pages/**/*.{ts,tsx}',
    './components/**/*.{ts,tsx}',
    './app/**/*.{ts,tsx}',
    './src/**/*.{ts,tsx}',
  ],
  theme: {
    container: {
      center: true,
      padding: "2rem",
      screens: {
        "2xl": "1400px",
      },
    },
    fontFamily: {
      logo: [
        'Ubuntu',
        'ui-sans-serif',
        'system-ui',
        'sans-serif',
        '"Apple Color Emoji"',
        '"Segoe UI Emoji"',
        '"Segoe UI Symbol"',
        '"Noto Color Emoji"',
      ],
      sans: [
        'ui-sans-serif',
        'system-ui',
        'sans-serif',
        '"Apple Color Emoji"',
        '"Segoe UI Emoji"',
        '"Segoe UI Symbol"',
        '"Noto Color Emoji"',
      ],
      serif: ['ui-serif', 'Georgia', 'Cambria', '"Times New Roman"', 'Times', 'serif'],
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
    extend: {
      colors: {
        ...agentPrismColors,
        "purple": {
          "50": "hsl(252, 82%, 95%)",
          "100": "hsl(252, 82%, 90%)",
          "200": "hsl(252, 82%, 80%)",
          "300": "hsl(252, 82%, 70%)",
          "400": "hsl(252, 82%, 60%)",
          "500": "hsl(252, 82%, 49%)",
          "600": "hsl(252, 82%, 40%)",
          "700": "hsl(252, 82%, 30%)",
          "800": "hsl(252, 82%, 20%)",
          "900": "hsl(252, 82%, 10%)"
        },
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        brand: "hsl(var(--brand))",
        primary: {
          DEFAULT: "hsl(var(--primary))",
          foreground: "hsl(var(--primary-foreground))",
        },
        secondary: {
          DEFAULT: "hsl(var(--secondary))",
          foreground: "hsl(var(--secondary-foreground))",
        },
        destructive: {
          DEFAULT: "hsl(var(--destructive))",
          foreground: "hsl(var(--destructive-foreground))",
        },
        muted: {
          DEFAULT: "hsl(var(--muted))",
          foreground: "hsl(var(--muted-foreground))",
        },
        accent: {
          DEFAULT: "hsl(var(--accent))",
          foreground: "hsl(var(--accent-foreground))",
        },
        popover: {
          DEFAULT: "hsl(var(--popover))",
          foreground: "hsl(var(--popover-foreground))",
        },
        card: {
          DEFAULT: "hsl(var(--card))",
          foreground: "hsl(var(--card-foreground))",
        },
        success: "hsl(var(--success))",
        danger: "hsl(var(--danger))",
      },
      borderRadius: {
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 4px)",
      },
      keyframes: {
        "accordion-down": {
          from: { height: 0 },
          to: { height: "var(--radix-accordion-content-height)" },
        },
        "accordion-up": {
          from: { height: "var(--radix-accordion-content-height)" },
          to: { height: 0 },
        },
        "flip": {
          to: {
            transform: "rotate(360deg)",
          },
        },
        "rotate": {
          to: {
            transform: "rotate(90deg)",
          },
        },
        "jiggle": {
          '0%, 100%': { transform: 'rotate(0deg)' },
          '25%': { transform: 'rotate(5deg)' },
          '75%': { transform: 'rotate(-5deg)' },
        },
      },
      animation: {
        "accordion-down": "accordion-down 0.2s ease-out",
        "accordion-up": "accordion-up 0.2s ease-out",
        "flip": "flip 6s infinite steps(2, end)",
        "rotate": "rotate 3s linear infinite both",
        "jiggle": 'jiggle 0.5s ease-in-out',

      },
    },
  },
  plugins: [require("tailwindcss-animate")],
}
