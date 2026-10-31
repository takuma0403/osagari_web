import { useLocation, useNavigate } from 'react-router'
import { AppHeader } from '../../../components/AppHeader/AppHeader'
import { LegalDocument } from '../components/LegalDocument/LegalDocument'
import { legalTitles, type LegalDoc } from '../legalDocs'
import styles from './LegalPage.module.css'

/**
 * 利用規約・プライバシーポリシー（AUTH-1, FE-1）。ログインの有無にかかわらず見られる。
 * 戻るは開いた元の画面（ログイン・マイページなど）へ。URL を直接開いて戻り先がなければ商品一覧へ
 * （未ログインならガードがログイン画面へ送る）
 */
export function LegalPage({ doc }: { doc: LegalDoc }) {
  const location = useLocation()
  const navigate = useNavigate()
  const back = () => (location.key === 'default' ? navigate('/') : navigate(-1))

  return (
    <div className={styles.page}>
      <AppHeader title={legalTitles[doc]} onBack={back} />
      <LegalDocument doc={doc} className={styles.content} />
    </div>
  )
}
