import { defineConfig } from "vite";
import react from "@vitejs/plugin-react-swc";
import path from "path";

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => ({
  server: {
    host: "::",
    port: 5173,
    strictPort: false,
    proxy: {
      '/api': {
        target: 'http://localhost:8082',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, '')
      },
      '/health': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
      '/query': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
      '/ingest': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
      '/labels': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
      '/metrics': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
      '/stream': {
        target: 'ws://localhost:8082',
        changeOrigin: true,
        ws: true,
      },
    },
  },
  plugins: [react()].filter(Boolean),
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
}));
