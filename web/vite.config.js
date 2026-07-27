import { defineConfig } from "vite";

// Two entry points: the viewer (index.html) and the owner console (admin.html).
// VITE_API_BASE is injected at build time (set it in Cloudflare Pages env vars).
export default defineConfig({
  build: {
    rollupOptions: {
      input: {
        main: "index.html",
        admin: "admin.html",
      },
    },
  },
});