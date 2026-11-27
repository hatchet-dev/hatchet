import { create } from 'storybook/theming';

// Colours from the app's tokens in src/index.css and the HatchetLogo brand
// colours, as hex because Storybook's theme doesn't read CSS variables.
const shared = {
  brandTitle: 'Hatchet',
  // Back to the Introduction page in the same tab.
  brandUrl: './',
  brandTarget: '_self',
  fontBase:
    'ui-sans-serif, system-ui, sans-serif, "Apple Color Emoji", "Segoe UI Emoji"',
  fontCode: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
  appBorderRadius: 6,
  inputBorderRadius: 6,
};

const light = create({
  ...shared,
  base: 'light',
  brandImage: './hatchet-logo-light.svg',
  colorPrimary: '#001A38',
  colorSecondary: '#001A38',
  appBg: '#F8FAFC',
  appContentBg: '#FFFFFF',
  appPreviewBg: '#FFFFFF',
  appBorderColor: '#E2E8F0',
  textColor: '#020817',
  textMutedColor: '#64748B',
  textInverseColor: '#F8FAFC',
  barBg: '#FFFFFF',
  barTextColor: '#64748B',
  barSelectedColor: '#001A38',
  barHoverColor: '#001A38',
  inputBg: '#FFFFFF',
  inputBorder: '#E2E8F0',
  inputTextColor: '#020817',
});

// A deeper shade of the logo blue: Storybook puts white text on the darkened
// accent for the selected sidebar item, which the logo blue is too light for.
const dark = create({
  ...shared,
  base: 'dark',
  brandImage: './hatchet-logo-dark.svg',
  colorPrimary: '#3D8BFF',
  colorSecondary: '#3D8BFF',
  appBg: '#020817',
  appContentBg: '#03091C',
  appPreviewBg: '#03091C',
  appBorderColor: '#1E293B',
  textColor: '#F8FAFC',
  textMutedColor: '#94A3B8',
  textInverseColor: '#03091C',
  barBg: '#03091C',
  barTextColor: '#94A3B8',
  barSelectedColor: '#3D8BFF',
  barHoverColor: '#3D8BFF',
  inputBg: '#020817',
  inputBorder: '#1E293B',
  inputTextColor: '#F8FAFC',
});

// Read once on load: Storybook doesn't re-theme its UI on the fly, so an OS
// switch applies after a reload.
export const hatchetTheme = window.matchMedia('(prefers-color-scheme: dark)')
  .matches
  ? dark
  : light;
