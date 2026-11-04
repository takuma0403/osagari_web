import type { ReactNode } from 'react'
import type { To } from 'react-router'
import { AppHeader } from '../../../../components/AppHeader/AppHeader'
import { StepIndicator } from '../../../../components/StepIndicator/StepIndicator'
import styles from './AuthFormLayout.module.css'

type Props = {
  /** 戻るボタンの行き先 */
  backTo: To
  /** 戻り先へ渡す location.state（再確認からログインへ戻るときの開こうとしていたページ） */
  backState?: unknown
  /** 手順の位置（STEP n / total）。再確認など手順のない画面では省く */
  step?: { current: number; total: number }
  /** 見出しの上に置くアイコン（再確認の盾） */
  icon?: ReactNode
  title: string
  /** 見出しの下の説明 */
  lead: ReactNode
  children: ReactNode
}

/**
 * 新規登録・パスワード再設定・再確認の画面の枠（SignUp.dc.html, Code.dc.html, Password.dc.html, Recheck.dc.html）。
 * 白地に戻るボタンだけのヘッダー、STEP・見出し・説明、その下に入力とボタンを並べる。
 * PC 幅（1024px 以上）では、灰色の地の中央に最大 440px の白いカードとして置く（DS-1）。
 */
export function AuthFormLayout({ backTo, backState, step, icon, title, lead, children }: Props) {
  return (
    <div className={styles.page}>
      {/* PC 幅では白いカードにして画面の中央に置く。スマホでは枠を持たない（DS-1） */}
      <div className={styles.card}>
        <AppHeader backTo={backTo} backState={backState} bordered={false} />
        <div className={styles.body}>
          <div className={styles.intro}>
            {step && <StepIndicator current={step.current} total={step.total} />}
            {icon}
            <h1 className={styles.title}>{title}</h1>
            <p className={styles.lead}>{lead}</p>
          </div>
          {children}
        </div>
      </div>
    </div>
  )
}
