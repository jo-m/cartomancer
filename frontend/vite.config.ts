import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  build: {
    outDir: "../static",
    emptyOutDir: true,
  },
  server: {
    proxy: {
      "/app_config": "http://localhost:8080",
      "/sessions": "http://localhost:8080",
      "/tracks": "http://localhost:8080",
      "/tags": "http://localhost:8080",
      "/account": "http://localhost:8080",
      "/admin": "http://localhost:8080",
    },
  },
});
