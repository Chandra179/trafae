import path from "node:path"
import { fileURLToPath } from "node:url"

import react from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"
import { defineConfig } from "vitest/config"

const rootDirectory = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(rootDirectory, "./src"),
    },
    // vitest externalizes node modules; without this the react-router chunk
    // and react-dom can end up with different React module instances.
    dedupe: ["react", "react-dom"],
  },
  test: {
    environment: "jsdom",
    server: {
      deps: {
        // React ships as CJS; externalized, vitest's ESM importer gets a
        // separate instance from react-dom's require, and hooks throw
        // "reading 'useRef'". Inlining keeps one module instance.
        inline: ["react", "react-dom", "react-router", "react-router-dom"],
      },
    },
  },
  server: {
    port: 5174,
    proxy: {
      "/api": {
        target: "http://localhost:8081",
        changeOrigin: true,
        rewrite: (requestPath) => requestPath.replace(/^\/api/, ""),
      },
    },
  },
})
