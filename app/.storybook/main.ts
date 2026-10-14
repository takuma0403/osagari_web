import type { StorybookConfig } from '@storybook/react-vite'

// DS-3 の共通部品のカタログ。ローカルで部品を確かめるためだけに使い、公開はしない（ADR 0004）
const config: StorybookConfig = {
  stories: ['../src/**/*.stories.tsx'],
  addons: ['@storybook/addon-a11y'],
  framework: '@storybook/react-vite',
  core: {
    disableTelemetry: true,
  },
}

export default config
