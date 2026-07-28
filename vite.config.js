import { defineConfig } from "vite";

export default defineConfig({
  publicDir: "public",
  build: {
    outDir: "build",
    assetsDir: "static",
    emptyOutDir: true,
  },
});
