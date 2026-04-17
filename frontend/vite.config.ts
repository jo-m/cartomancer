import { defineConfig, type Plugin } from "vite"
import react from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"
import { marked } from "marked"

const cspContent = [
  "default-src 'self'",
  "script-src 'self'",
  "style-src 'self' 'unsafe-inline'",
  "img-src 'self' data: blob: https://wmts.geo.admin.ch",
  "connect-src 'self' https://wmts.geo.admin.ch",
  "font-src 'self'",
  "object-src 'none'",
  "base-uri 'self'",
].join("; ")

/** Vite plugin that injects a Content-Security-Policy meta tag in production builds. */
function cspPlugin(): Plugin {
  const meta = `<meta http-equiv="Content-Security-Policy" content="${cspContent}">`
  return {
    name: "csp",
    transformIndexHtml: {
      order: "post",
      handler(html, ctx) {
        if (ctx.bundle) {
          return html.replace("<head>", `<head>\n    ${meta}`)
        }
      },
    },
  }
}

/** Vite plugin that compiles .md files to HTML strings at build time. */
function markdownPlugin(): Plugin {
  return {
    name: "markdown",
    transform(src, id) {
      if (id.endsWith(".md")) {
        const html = marked(src) as string
        return { code: `export default ${JSON.stringify(html)}` }
      }
    },
  }
}

export default defineConfig({
  plugins: [cspPlugin(), markdownPlugin(), react(), tailwindcss()],
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
})
