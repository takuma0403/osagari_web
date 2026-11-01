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
  /** 戻るボタンを「閉じる」（×）にする。出品のように、入力をやめて元の画面へ戻る画面で使う */
  closable?: boolean
  /** 暗い地（QRの読み取りのカメラ画面）の上に置くときは dark。文字と戻るボタンを白にする */
  tone?: 'default' | 'dark'
}

/** 画面上部のヘッダー（DS-3）。高さ56px、戻るボタンは 44px */
export function AppHeader({
  title,
  backTo,
  backState,
  onBack,
  right,
  bordered = true,
  closable,
  tone = 'default',
}: Props) {
  const back = <Icon name={closable ? 'close' : 'back'} />
  const backLabel = closable ? '閉じる' : '戻る'
  return (
    <header
      className={[styles.header, bordered && styles.bordered, tone === 'dark' && styles.dark]
        .filter(Boolean)
        .join(' ')}
    >
      {backTo !== undefined ? (
        <Link to={backTo} state={backState} className={styles.back} aria-label={backLabel}>
          {back}
        </Link>
      ) : (
        onBack && (
          <button type="button" className={styles.back} aria-label={backLabel} onClick={onBack}>
            {back}
          </button>
        )
      )}
      {title && <h1 className={styles.title}>{title}</h1>}
      {right && <div className={styles.right}>{right}</div>}
    </header>
  )
}
