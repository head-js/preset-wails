import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const port = Number(process.env.WAILS_VITE_PORT) || 5173;

export default defineConfig({
  plugins: [react()],
  root: "src",
  clearScreen: false,
  server: {
    host: "127.0.0.1",
    port,
    strictPort: true,
  },
});
