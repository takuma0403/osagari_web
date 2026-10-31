import { Suspense, use, type MouseEvent } from 'react'
import { useLocation, useNavigate } from 'react-router'
import { AppHeader } from '../../../components/AppHeader/AppHeader'
import styles from './LegalPage.module.css'

type Doc = 'terms' | 'privacy'

const TITLES: Record<Doc, string> = {
  terms: '利用規約',
  privacy: 'プライバシーポリシー',
}

// 本文は docs/legal の正本をビルド時に HTML にしたもの（vite/legalMarkdown.ts）。
// めったに開かない画面なので、メインのバンドルに入れずに開いたときに読み込む
const LOADERS: Record<Doc, () => Promise<{ default: string }>> = {
  terms: () => import('virtual:legal/terms'),
  privacy: () => import('virtual:legal/privacy'),
}

const bodies = new Map<Doc, Promise<string>>()

function loadBody(doc: Doc): Promise<string> {
  let body = bodies.get(doc)
  if (!body) {
    body = LOADERS[doc]().then((module) => module.default)
    bodies.set(doc, body)
  }
  return body
}

function LegalBody({ doc }: { doc: Doc }) {
  const navigate = useNavigate()
  // 中身はリポジトリの正本だけで、利用者の入力は入らない
  const html = use(loadBody(doc))

  // 本文中の /terms・/privacy へのリンクは、ページを読み込み直さずに SPA の中で開く
  const handleClick = (event: MouseEvent<HTMLDivElement>) => {
    if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
      return
    }
    const href = (event.target as Element).closest('a')?.getAttribute('href')
    if (!href?.startsWith('/')) return
    event.preventDefault()
    navigate(href)
  }

  return (
    <div className={styles.body} onClick={handleClick} dangerouslySetInnerHTML={{ __html: html }} />
  )
}

/**
 * 利用規約・プライバシーポリシー（AUTH-1, FE-1）。ログインの有無にかかわらず見られる。
 * 戻るは開いた元の画面（ログイン・新規登録・マイページなど）へ。URL を直接開いて戻り先がなければ商品一覧へ
 * （未ログインならガードがログイン画面へ送る）
 */
export function LegalPage({ doc }: { doc: Doc }) {
  const location = useLocation()
  const navigate = useNavigate()
  const back = () => (location.key === 'default' ? navigate('/') : navigate(-1))

  return (
    <div className={styles.page}>
      <AppHeader title={TITLES[doc]} onBack={back} />
      <Suspense>
        <LegalBody doc={doc} />
      </Suspense>
    </div>
  )
}
