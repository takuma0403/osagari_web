import type { StorybookConfig } from '@storybook/react-vite'

// DS-3 の共通部品のカタログ。ローカルで部品を確かめるために使い、公開はしない（ADR 0004）。
// 全 story の描画と a11y は addon-vitest で PR の CI でも検査する（npm run test:storybook、ADR 0005）
const config: StorybookConfig = {
  stories: ['../src/**/*.stories.tsx'],
  addons: ['@storybook/addon-a11y', '@storybook/addon-vitest'],
  framework: '@storybook/react-vite',
  core: {
    disableTelemetry: true,
  },
}

export default config
