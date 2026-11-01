import { useState, type FormEvent } from 'react'
import { Icon } from '../../../../components/Icon/Icon'
import { SEARCH_TEXT_MAX } from '../../searchConditions'
import styles from './SearchField.module.css'

type Props = {
  /** 最初に入れておくキーワード。URL の条件が変わったら key を変えて入れ直す */
  defaultValue?: string
  placeholder?: string
  /** 確定（Enter・キーボードの検索）したときに、前後の空白を除いたキーワードで呼ぶ */
  onSearch: (keyword: string) => void
  className?: string
}

/** 角丸の検索欄（Main・Search）。虫眼鏡アイコン＋キーワード入力。確定は Enter で行う */
export function SearchField({ defaultValue = '', placeholder, onSearch, className }: Props) {
  const [value, setValue] = useState(defaultValue)

  const submit = (e: FormEvent) => {
    e.preventDefault()
    onSearch(value.trim())
  }

  return (
    <form
      role="search"
      className={[styles.form, className].filter(Boolean).join(' ')}
      onSubmit={submit}
    >
      <label className={styles.box}>
        <Icon name="search" size={18} className={styles.icon} />
        <span className="visually-hidden">キーワード</span>
        <input
          type="search"
          enterKeyHint="search"
          className={styles.input}
          value={value}
          placeholder={placeholder}
          maxLength={SEARCH_TEXT_MAX}
          onChange={(e) => setValue(e.target.value)}
        />
      </label>
    </form>
  )
}
