import { defineConfig, type Plugin } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { marked } from "marked";

/** Vite plugin that compiles .md files to HTML strings at build time. */
function markdownPlugin(): Plugin {
  return {
    name: "markdown",
    transform(src, id) {
      if (id.endsWith(".md")) {
        const html = marked(src) as string;
        return { code: `export default ${JSON.stringify(html)}` };
      }
    },
  };
}

export default defineConfig({
  plugins: [markdownPlugin(), react(), tailwindcss()],
  build: {
    outDir: "../static",
    emptyOutDir: true,
  },
  server: {
    proxy: {
      "/api": "http://localhost:8080",
      "/robots.txt": "http://localhost:8080",
    },
  },
});
