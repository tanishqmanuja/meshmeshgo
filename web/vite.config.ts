import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => ({
  plugins: [react()],
  server: {
    host: true,
  },
  build: {
    outDir: "../managerui",
    emptyOutDir: true,
    sourcemap: mode === "development",
  },
  base: "./",
}));
