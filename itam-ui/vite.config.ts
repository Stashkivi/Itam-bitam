import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { '@': path.resolve(__dirname, 'src') },
  },
  server: {
    port: 5173,
    proxy: {
      '/v1': {
        target:    'https://localhost:8443',
        secure:    false,
        changeOrigin: true,
        ws: true,   // proxies /v1/ws WebSocket upgrades
      },
    },
  },
});
