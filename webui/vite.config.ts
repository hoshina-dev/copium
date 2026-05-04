import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Vite serves the SPA on http://localhost:5173 during development and
// proxies API + docs traffic to the Go backend on :8081 so that the same
// `fetch("/api/v1/...")` calls work in dev and in the embedded production
// build (where Fiber serves both the assets and the API).
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      "/api": "http://localhost:8081",
      "/healthz": "http://localhost:8081",
      "/readyz": "http://localhost:8081",
      "/swagger": "http://localhost:8081",
      "/scalar": "http://localhost:8081",
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
    sourcemap: true,
  },
});
