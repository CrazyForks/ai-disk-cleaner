import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'node:path'
import * as fs from 'node:fs'

type Project = {
  version: string
}

const project = JSON.parse(
  fs.readFileSync(path.resolve(__dirname, '../project.json'), {
    encoding: 'utf-8',
  }),
) as Project

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
      '@backend': path.resolve(__dirname, 'wailsjs'),
    },
  },
  define: {
    __APP_VERSION__: project.version,
  },
})
