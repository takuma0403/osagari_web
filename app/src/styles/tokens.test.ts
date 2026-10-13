/// <reference types="node" />
import { readFileSync } from 'node:fs'

const read = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')
const designDoc = read('../../../docs/design/README.md')
const tokensCss = read('./tokens.css')

// DS-2: tokens.css は docs/design/README.md の表と同じ名前・値の CSS 変数だけを持つ
function tokensFromDoc(doc: string): Map<string, string> {
  const section = doc.slice(doc.indexOf('## DS-2'), doc.indexOf('## DS-3'))
  const map = new Map<string, string>()
  // 表の行：| `--name` | `value` | 用途 |
  for (const m of section.matchAll(/^\| `(--[a-z0-9-]+)` \| `([^`]+)`/gm)) {
    map.set(m[1], m[2])
  }
  // 用途欄で定義されているもの：「24px = `--space-page-form`」
  for (const m of section.matchAll(/(\d+px) = `(--[a-z0-9-]+)`/g)) {
    map.set(m[2], m[1])
  }
  // カテゴリ色は本文の「教科書 `#DDF0F0`／…」と「`--color-cat-…` として定義する」から対応づける
  const catColors = [...section.matchAll(/(?:教科書|家電|家具|ゲーム) `(#[0-9A-F]{6})`/g)].map(
    (m) => m[1],
  )
  const catNames = [...section.matchAll(/`(--color-cat-[a-z]+)`/g)].map((m) => m[1])
  catNames.forEach((name, i) => map.set(name, catColors[i]))
  return map
}

function tokensFromCss(css: string): Map<string, string> {
  return new Map([...css.matchAll(/^\s*(--[a-z0-9-]+):\s*([^;]+);/gm)].map((m) => [m[1], m[2]]))
}

// Prettier が 16 進カラーを小文字にするため、大文字小文字を区別せずに比べる
const normalize = (map: Map<string, string>) =>
  Object.fromEntries([...map].map(([k, v]) => [k, v.toLowerCase()]))

describe('tokens.css', () => {
  it('DS-2 の表と名前・値が一致する', () => {
    const doc = tokensFromDoc(designDoc)
    const css = tokensFromCss(tokensCss)
    expect(doc.size).toBeGreaterThan(40)
    expect(normalize(css)).toEqual(normalize(doc))
  })
})
