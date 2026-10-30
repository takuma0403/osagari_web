/// <reference types="vitest/config" />
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin'
import react from '@vitejs/plugin-react'
import { playwright } from '@vitest/browser-playwright'
import { defineConfig } from 'vite'
import { legalMarkdown } from './vite/legalMarkdown.ts'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), legalMarkdown()],
  server: {
    // 本番は Firebase Hosting の rewrites で /api/** を Cloud Run に転送する（ARCH）。ローカルは server の compose（:8080）へ
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  test: {
    projects: [
      // 単体テスト（jsdom）。npm test
      {
        extends: true,
        test: {
          name: 'unit',
          include: ['src/**/*.test.{ts,tsx}', 'vite/**/*.test.ts'],
          environment: 'jsdom',
          globals: true,
          setupFiles: ['./src/test/setup.ts'],
        },
      },
      // 全 story の描画と a11y 検査（Chromium）。npm run test:storybook（ADR 0005）
      {
        extends: true,
        plugins: [storybookTest({ configDir: '.storybook' })],
        test: {
          name: 'storybook',
          browser: {
            enabled: true,
            headless: true,
            provider: playwright(),
            instances: [{ browser: 'chromium' }],
          },
        },
      },
    ],
  },
})
