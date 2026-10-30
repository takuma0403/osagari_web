// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { legalMarkdownToHtml } from './legalMarkdown.ts'

const markdown = `# 利用規約

> 利用規約の正本です。画面（\`/terms\`）にはこの本文を載せます。
> 本文中の「（未定）」は、決まるまで公開できない箇所です。

この規約は、本サービスの利用条件を定めるものです。

### 第１条（本サービス）

1. 扱いは [プライバシーポリシー](privacy.md) に定めます。
2. 規約は [利用規約](terms.md)、未決は [TECH-OPEN](../tech/open-questions.md)。

| 情報 | 取得するとき |
| --- | --- |
| 学籍番号 | 新規登録 |
`

describe('legalMarkdownToHtml', () => {
  const html = legalMarkdownToHtml(markdown)

  it('先頭の見出しと docs 向けの注記は載せない', () => {
    expect(html).not.toContain('<h1')
    expect(html).not.toContain('正本です')
    expect(html).not.toContain('<blockquote')
    expect(html).toMatch(/^<p>この規約は/)
  })

  it('正本どうしのリンクは画面のパスにする', () => {
    expect(html).toContain('<a href="/privacy">プライバシーポリシー</a>')
    expect(html).toContain('<a href="/terms">利用規約</a>')
  })

  it('ほかの docs へのリンクは外して文字だけ残す', () => {
    expect(html).not.toContain('open-questions')
    expect(html).toContain('未決は TECH-OPEN')
  })

  it('見出し・番号付きの条文・表を変換する', () => {
    expect(html).toContain('<h3>第１条（本サービス）</h3>')
    expect(html).toContain('<ol>')
    expect(html).toContain('<th>情報</th>')
    expect(html).toContain('<td>学籍番号</td>')
  })
})
