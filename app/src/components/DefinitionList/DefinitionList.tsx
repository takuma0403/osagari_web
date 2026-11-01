import { Fragment, type ReactNode } from 'react'
import styles from './DefinitionList.module.css'

export type DefinitionItem = {
  term: string
  /** 学籍番号・金額などは利用側で Outfit にする */
  description: ReactNode
}

type Props = {
  items: DefinitionItem[]
  /** md 14px（商品詳細）／sm 13px（確認の枠など） */
  size?: 'md' | 'sm'
  /** 地の色の枠に入れる */
  boxed?: boolean
  className?: string
}

/** 項目名（muted）と値を2列で並べる `<dl>`（DS-3） */
export function DefinitionList({ items, size = 'md', boxed, className }: Props) {
  return (
    <dl
      className={[styles.list, size === 'sm' && styles.sm, boxed && styles.boxed, className]
        .filter(Boolean)
        .join(' ')}
    >
      {items.map(({ term, description }) => (
        <Fragment key={term}>
          <dt className={styles.term}>{term}</dt>
          <dd>{description}</dd>
        </Fragment>
      ))}
    </dl>
  )
}
