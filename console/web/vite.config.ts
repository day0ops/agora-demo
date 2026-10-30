import { defineConfig } from 'vite';
import { resolve } from 'path';

export default defineConfig({
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    rollupOptions: {
      input: {
        index: resolve(__dirname, 'index.html'),
        shop: resolve(__dirname, 'shop.html'),
        sell: resolve(__dirname, 'sell.html'),
        ops: resolve(__dirname, 'ops.html'),
      },
    },
  },
  server: { proxy: { '/api': 'http://localhost:8090' } },
});
