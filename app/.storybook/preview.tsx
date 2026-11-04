import type { Preview } from '@storybook/react-vite'
import { MemoryRouter } from 'react-router'
import '../src/styles/tokens.css'
import '../src/styles/global.css'

const preview: Preview = {
  // Link・NavLink を使う部品があるため、すべての story をルーターで包む。いまのパスは parameters.path で変えられる
  decorators: [
    (Story, { parameters }) => (
      <MemoryRouter initialEntries={[parameters.path ?? '/']}>
        <Story />
      </MemoryRouter>
    ),
  ],
  parameters: {
    layout: 'padded',
    // DS-1: 利用者向け画面は 390px のスマホが基準。640px・1024px で PC 幅の表示に切り替わるので、その確認用も置く
    viewport: {
      options: {
        mobile390: {
          name: 'スマホ（390×844）',
          styles: { width: '390px', height: '844px' },
          type: 'mobile',
        },
        tablet640: {
          name: 'タブレット（640×1024）',
          styles: { width: '640px', height: '1024px' },
          type: 'tablet',
        },
        pc1280: {
          name: 'PC（1280×800）',
          styles: { width: '1280px', height: '800px' },
          type: 'desktop',
        },
      },
    },
    a11y: {
      // アクセシビリティ違反は Storybook の画面上でエラーとして示し、story のテスト（CI）では失敗にする（DS-1）
      test: 'error',
    },
  },
  initialGlobals: {
    viewport: { value: 'mobile390', isRotated: false },
  },
}

export default preview
