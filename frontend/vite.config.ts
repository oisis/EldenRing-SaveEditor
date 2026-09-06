import { lingui, linguiTransformerBabelPreset } from "@lingui/vite-plugin";
import babel from "@rolldown/plugin-babel";
import { vanillaExtractPlugin } from "@vanilla-extract/vite-plugin";
import react from "@vitejs/plugin-react";
import { defineConfig, type Plugin } from "vitest/config";

/**
 * `wails dev` proxies every request to this dev server first and only falls back
 * to the Go catalog asset handler when Vite answers 404. Vite's SPA fallback
 * would answer /catalog-assets/ with index.html, so item icons and appearance
 * previews would arrive as HTML. Answering 404 here keeps the single validated
 * asset contract reachable in development; production serves the same URLs from
 * the embedded AssetServer.
 */
function catalogAssetsFallthrough(): Plugin {
  return {
    name: "saveforge-catalog-assets-fallthrough",
    apply: "serve",
    configureServer(server) {
      server.middlewares.use((request, response, next) => {
        if (!request.url?.startsWith("/catalog-assets/")) {
          next();
          return;
        }
        response.statusCode = 404;
        response.end();
      });
    },
  };
}

export default defineConfig({
  plugins: [
    react(),
    lingui(),
    babel({ presets: [linguiTransformerBabelPreset()] }),
    vanillaExtractPlugin(),
    catalogAssetsFallthrough(),
  ],
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    include: ["src/**/*.test.ts", "src/**/*.test.tsx"],
  },
});
