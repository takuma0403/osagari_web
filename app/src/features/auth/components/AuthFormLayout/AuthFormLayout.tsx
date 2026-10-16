import type { ReactNode } from 'react'
import type { To } from 'react-router'
import { AppHeader } from '../../../../components/AppHeader/AppHeader'
import { StepIndicator } from '../../../../components/StepIndicator/StepIndicator'
import styles from './AuthFormLayout.module.css'

type Props = {
  /** 戻るボタンの行き先 */
  backTo: To
  /** 手順の位置（STEP n / total）。再確認など手順のない画面では省く */
  step?: { current: number; total: number }
  title: string
  /** 見出しの下の説明 */
  lead: ReactNode
  children: ReactNode
}

/**
 * 新規登録・パスワード再設定・再確認の画面の枠（SignUp.dc.html, Code.dc.html, Password.dc.html）。
 * 白地に戻るボタンだけのヘッダー、STEP・見出し・説明、その下に入力とボタンを並べる。
 */
export function AuthFormLayout({ backTo, step, title, lead, children }: Props) {
  return (
    <div className={styles.page}>
      <AppHeader backTo={backTo} bordered={false} />
      <div className={styles.body}>
        <div className={styles.intro}>
          {step && <StepIndicator current={step.current} total={step.total} />}
          <h1 className={styles.title}>{title}</h1>
          <p className={styles.lead}>{lead}</p>
        </div>
        {children}
      </div>
    </div>
  )
}
