import path from "node:path";

import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
    },
  },
  server: {
    host: "0.0.0.0",
    allowedHosts: [".e2b.app"],
    proxy: {
      "/api": {
        target: "http://127.0.0.1:8080",
        changeOrigin: true,
      },
    },
  },
  preview: {
    host: "0.0.0.0",
    allowedHosts: [".e2b.app"],
  },
  build: {
    // Keep the initial dashboard payload small as feature routes and UI
    // primitives grow; vendor chunks remain long-cacheable in production.
    rolldownOptions: {
      output: {
        manualChunks(moduleId) {
          if (!moduleId.includes("node_modules")) {
            return undefined;
          }
          if (moduleId.includes("react-router") || moduleId.includes("react-dom") || moduleId.includes("/react/")) {
            return "react-vendor";
          }
          if (moduleId.includes("@radix-ui") || moduleId.includes("lucide-react")) {
            return "ui-vendor";
          }
          if (moduleId.includes("@tanstack") || moduleId.includes("zustand")) {
            return "state-vendor";
          }
          return "utility-vendor";
        },
      },
    },
  },
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"],
    globals: false,
  },
});
