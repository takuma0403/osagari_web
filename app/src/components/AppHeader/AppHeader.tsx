import type { ReactNode } from 'react'
import { Link, type To } from 'react-router'
import { Icon } from '../Icon/Icon'
import styles from './AppHeader.module.css'

type Props = {
  /** 画面のタイトル（h1）。認証コード入力など、タイトルを本文に置く画面では省く */
  title?: string
  /** 戻るボタンの行き先。onBack と両方なければ戻るボタンを出さない */
  backTo?: To
  /** 戻り先へ渡す location.state */
  backState?: unknown
  /** 戻るボタンを押したときの処理（行き先が決まらないとき。例 navigate(-1)） */
  onBack?: () => void
  /** 右側に置くもの（バッジ・アイコンボタンなど） */
  right?: ReactNode
  /** 白地と下線を付ける。フォーム系の画面（地が白）では false */
  bordered?: boolean
}

/** 画面上部のヘッダー（DS-3）。高さ56px、戻るボタンは 44px */
export function AppHeader({ title, backTo, backState, onBack, right, bordered = true }: Props) {
  const back = <Icon name="back" />
  return (
    <header className={[styles.header, bordered && styles.bordered].filter(Boolean).join(' ')}>
      {backTo !== undefined ? (
        <Link to={backTo} state={backState} className={styles.back} aria-label="戻る">
          {back}
        </Link>
      ) : (
        onBack && (
          <button type="button" className={styles.back} aria-label="戻る" onClick={onBack}>
            {back}
          </button>
        )
      )}
      {title && <h1 className={styles.title}>{title}</h1>}
      {right && <div className={styles.right}>{right}</div>}
    </header>
  )
}
