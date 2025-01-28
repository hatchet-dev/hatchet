// vite.config.ts
import { sentryVitePlugin } from "file:///Users/gabrielruttner/dev/hatchet-all/repos/oss/frontend/app/node_modules/.pnpm/@sentry+vite-plugin@2.16.0/node_modules/@sentry/vite-plugin/dist/esm/index.mjs";
import path from "path";
import react from "file:///Users/gabrielruttner/dev/hatchet-all/repos/oss/frontend/app/node_modules/.pnpm/@vitejs+plugin-react@4.2.1_vite@5.2.7_@types+node@20.12.2_/node_modules/@vitejs/plugin-react/dist/index.mjs";
import { defineConfig } from "file:///Users/gabrielruttner/dev/hatchet-all/repos/oss/frontend/app/node_modules/.pnpm/vite@5.2.7_@types+node@20.12.2/node_modules/vite/dist/node/index.js";
var __vite_injected_original_dirname = "/Users/gabrielruttner/dev/hatchet-all/repos/oss/frontend/app";
var vite_config_default = defineConfig({
  plugins: [
    react(),
    sentryVitePlugin({
      org: "hatchet",
      project: "frontend-react"
    })
  ],
  resolve: {
    alias: {
      "@": path.resolve(__vite_injected_original_dirname, "./src")
    }
  },
  build: {
    sourcemap: true
  }
});
export {
  vite_config_default as default
};
//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsidml0ZS5jb25maWcudHMiXSwKICAic291cmNlc0NvbnRlbnQiOiBbImNvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9kaXJuYW1lID0gXCIvVXNlcnMvZ2FicmllbHJ1dHRuZXIvZGV2L2hhdGNoZXQtYWxsL3JlcG9zL29zcy9mcm9udGVuZC9hcHBcIjtjb25zdCBfX3ZpdGVfaW5qZWN0ZWRfb3JpZ2luYWxfZmlsZW5hbWUgPSBcIi9Vc2Vycy9nYWJyaWVscnV0dG5lci9kZXYvaGF0Y2hldC1hbGwvcmVwb3Mvb3NzL2Zyb250ZW5kL2FwcC92aXRlLmNvbmZpZy50c1wiO2NvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9pbXBvcnRfbWV0YV91cmwgPSBcImZpbGU6Ly8vVXNlcnMvZ2FicmllbHJ1dHRuZXIvZGV2L2hhdGNoZXQtYWxsL3JlcG9zL29zcy9mcm9udGVuZC9hcHAvdml0ZS5jb25maWcudHNcIjtpbXBvcnQgeyBzZW50cnlWaXRlUGx1Z2luIH0gZnJvbSAnQHNlbnRyeS92aXRlLXBsdWdpbic7XG5pbXBvcnQgcGF0aCBmcm9tICdwYXRoJztcbmltcG9ydCByZWFjdCBmcm9tICdAdml0ZWpzL3BsdWdpbi1yZWFjdCc7XG5pbXBvcnQgeyBkZWZpbmVDb25maWcgfSBmcm9tICd2aXRlJztcblxuZXhwb3J0IGRlZmF1bHQgZGVmaW5lQ29uZmlnKHtcbiAgcGx1Z2luczogW1xuICAgIHJlYWN0KCksXG4gICAgc2VudHJ5Vml0ZVBsdWdpbih7XG4gICAgICBvcmc6ICdoYXRjaGV0JyxcbiAgICAgIHByb2plY3Q6ICdmcm9udGVuZC1yZWFjdCcsXG4gICAgfSksXG4gIF0sXG4gIHJlc29sdmU6IHtcbiAgICBhbGlhczoge1xuICAgICAgJ0AnOiBwYXRoLnJlc29sdmUoX19kaXJuYW1lLCAnLi9zcmMnKSxcbiAgICB9LFxuICB9LFxuICBidWlsZDoge1xuICAgIHNvdXJjZW1hcDogdHJ1ZSxcbiAgfSxcbn0pO1xuIl0sCiAgIm1hcHBpbmdzIjogIjtBQUFzVyxTQUFTLHdCQUF3QjtBQUN2WSxPQUFPLFVBQVU7QUFDakIsT0FBTyxXQUFXO0FBQ2xCLFNBQVMsb0JBQW9CO0FBSDdCLElBQU0sbUNBQW1DO0FBS3pDLElBQU8sc0JBQVEsYUFBYTtBQUFBLEVBQzFCLFNBQVM7QUFBQSxJQUNQLE1BQU07QUFBQSxJQUNOLGlCQUFpQjtBQUFBLE1BQ2YsS0FBSztBQUFBLE1BQ0wsU0FBUztBQUFBLElBQ1gsQ0FBQztBQUFBLEVBQ0g7QUFBQSxFQUNBLFNBQVM7QUFBQSxJQUNQLE9BQU87QUFBQSxNQUNMLEtBQUssS0FBSyxRQUFRLGtDQUFXLE9BQU87QUFBQSxJQUN0QztBQUFBLEVBQ0Y7QUFBQSxFQUNBLE9BQU87QUFBQSxJQUNMLFdBQVc7QUFBQSxFQUNiO0FBQ0YsQ0FBQzsiLAogICJuYW1lcyI6IFtdCn0K
