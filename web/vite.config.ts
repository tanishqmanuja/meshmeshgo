import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => ({
  plugins: [react()],
  server: {
    host: true,
  },
  build: {
    outDir: "../mangerui",
    sourcemap: mode === "development",
  },
  base: "./",
}));
