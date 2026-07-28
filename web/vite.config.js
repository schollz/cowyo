import { defineConfig } from "vite";

export default defineConfig({
  publicDir: "public",
  build: {
    outDir: "../internal/site/build",
    assetsDir: "static",
    emptyOutDir: true,
  },
});
