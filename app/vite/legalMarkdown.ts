import { readFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import { Marked } from 'marked'
import type { Plugin } from 'vite'

// 利用規約・プライバシーポリシーの本文の正本（docs/legal）。画面は /terms・/privacy（FE-1）
const DOCS = ['terms', 'privacy'] as const
const VIRTUAL_PREFIX = 'virtual:legal/'

// 正本どうしのリンクはアプリの画面へ向ける。それ以外の docs へのリンクは画面では開けないので文字だけ残す
const APP_LINKS: Record<string, string> = {
  'terms.md': '/terms',
  'privacy.md': '/privacy',
}

const marked = new Marked({
  gfm: true,
  renderer: {
    link({ href, tokens }) {
      if (/^https?:\/\//.test(href)) return false
      const text = this.parser.parseInline(tokens)
      const to = APP_LINKS[href]
      return to ? `<a href="${to}">${text}</a>` : text
    },
  },
})

/**
 * 正本の Markdown を画面に載せる HTML にする。
 * 先頭の見出し（画面ではヘッダーに出す）と、そのあとの引用（docs 向けの注記）は載せない。
 */
export function legalMarkdownToHtml(markdown: string): string {
  const body = markdown.replace(/^# .*\n+(?:>.*\n)*\n*/, '')
  return marked.parse(body, { async: false })
}

/**
 * `virtual:legal/terms`・`virtual:legal/privacy` を、正本をビルド時に HTML にした文字列として読み込ませる。
 * 実行時に Markdown の変換を持ち込まないため（FE-1）
 */
export function legalMarkdown(): Plugin {
  return {
    name: 'legal-markdown',
    resolveId(id) {
      return id.startsWith(VIRTUAL_PREFIX) ? `\0${id}` : undefined
    },
    async load(id) {
      if (!id.startsWith(`\0${VIRTUAL_PREFIX}`)) return undefined
      const name = id.slice(VIRTUAL_PREFIX.length + 1)
      if (!(DOCS as readonly string[]).includes(name)) {
        this.error(`docs/legal に ${name}.md はありません（${DOCS.join('・')} のみ）`)
      }
      const file = fileURLToPath(new URL(`../../docs/legal/${name}.md`, import.meta.url))
      this.addWatchFile(file)
      const html = legalMarkdownToHtml(await readFile(file, 'utf8'))
      return `export default ${JSON.stringify(html)}`
    },
  }
}
