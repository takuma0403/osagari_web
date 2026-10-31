import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, userEvent, within } from 'storybook/test'
import { TermsConsent } from './TermsConsent'

const meta = {
  title: '認証/TermsConsent',
  component: TermsConsent,
  args: { checked: false, onChange: () => {} },
  // チェックの状態は画面が持つので、story でも持たせる
  render: function Render(args) {
    const [checked, setChecked] = useState(args.checked)
    return <TermsConsent checked={checked} onChange={setChecked} />
  },
} satisfies Meta<typeof TermsConsent>

export default meta
type Story = StoryObj<typeof meta>

/** まだどちらも開いていない（チェックできない） */
export const Default: Story = {}

/** 利用規約をシートで開いたところ */
export const SheetOpen: Story = {
  play: async ({ canvasElement }) => {
    await userEvent.click(within(canvasElement).getByRole('button', { name: '利用規約' }))
  },
}

/** 両方を開いて同意にチェックしたところ */
export const Agreed: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const body = within(document.body)
    for (const name of ['利用規約', 'プライバシーポリシー']) {
      await userEvent.click(canvas.getByRole('button', { name }))
      await userEvent.click(await body.findByRole('button', { name: '閉じる' }))
    }
    await userEvent.click(canvas.getByRole('checkbox', { name: /同意します/ }))
    await expect(canvas.getByRole('checkbox', { name: /同意します/ })).toBeChecked()
  },
}
