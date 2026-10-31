import { Suspense, use, type MouseEvent } from 'react'
import { useNavigate } from 'react-router'
import type { LegalDoc } from '../../legalDocs'
import styles from './LegalDocument.module.css'

/** 本文中のリンクのうち、規約の文書を指すパスと文書の対応 */
const docPaths: Record<string, LegalDoc> = {
  '/terms': 'terms',
  '/privacy': 'privacy',
}

// 本文は docs/legal の正本をビルド時に HTML にしたもの（vite/legalMarkdown.ts）。
// めったに開かない画面なので、メインのバンドルに入れずに開いたときに読み込む
const LOADERS: Record<LegalDoc, () => Promise<{ default: string }>> = {
  terms: () => import('virtual:legal/terms'),
  privacy: () => import('virtual:legal/privacy'),
}

const bodies = new Map<LegalDoc, Promise<string>>()

function loadBody(doc: LegalDoc): Promise<string> {
  let body = bodies.get(doc)
  if (!body) {
    body = LOADERS[doc]().then((module) => module.default)
    bodies.set(doc, body)
  }
  return body
}

type Props = {
  doc: LegalDoc
  /**
   * 本文中の /terms・/privacy へのリンクを押したとき。省くとその画面へ SPA の中で移る。
   * 新規登録のシートのように、画面を移らずに表示を切り替えるときに渡す
   */
  onOpenDoc?: (doc: LegalDoc) => void
  className?: string
}

function LegalBody({ doc, onOpenDoc, className }: Props) {
  const navigate = useNavigate()
  // 中身はリポジトリの正本だけで、利用者の入力は入らない
  const html = use(loadBody(doc))

  // 本文中のアプリ内のリンクは、ページを読み込み直さずに SPA の中で開く
  const handleClick = (event: MouseEvent<HTMLDivElement>) => {
    if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
      return
    }
    const href = (event.target as Element).closest('a')?.getAttribute('href')
    if (!href?.startsWith('/')) return
    event.preventDefault()
    const linked = docPaths[href]
    if (linked && onOpenDoc) onOpenDoc(linked)
    else navigate(href)
  }

  return (
    <div
      className={[styles.body, className].filter(Boolean).join(' ')}
      onClick={handleClick}
      dangerouslySetInnerHTML={{ __html: html }}
    />
  )
}

/** 利用規約・プライバシーポリシーの本文（AUTH-1, FE-1）。読み込み中は何も出さない */
export function LegalDocument(props: Props) {
  return (
    <Suspense>
      <LegalBody {...props} />
    </Suspense>
  )
}
