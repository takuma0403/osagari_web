import { useId, useState, type FormEvent } from 'react'
import { BottomSheet } from '../../../../components/BottomSheet/BottomSheet'
import { Button } from '../../../../components/Button/Button'
import { Chip } from '../../../../components/Chip/Chip'
import { TextField } from '../../../../components/TextField/TextField'
import {
  campusOptions,
  categoryOptions,
  conditionOptions,
  type Option,
} from '../../../../lib/itemOptions'
import { SEARCH_TEXT_MAX, type ItemSearchConditions } from '../../searchConditions'
import styles from './SearchFilterSheet.module.css'

/** シートで選ぶ条件（キーワードは検索欄で入れるので含めない） */
export type FilterConditions = Omit<ItemSearchConditions, 'keyword'>

type Props = {
  /** いまの条件。開いたときの初期値にする */
  conditions: FilterConditions
  /** 「この条件で検索」。指定のない条件は含めない */
  onApply: (conditions: FilterConditions) => void
  onClose: () => void
}

type Draft = {
  category: string
  courseName: string
  /** 数字だけの文字列 */
  priceMin: string
  priceMax: string
  campus: string
  condition: string
}

const emptyDraft: Draft = {
  category: '',
  courseName: '',
  priceMin: '',
  priceMax: '',
  campus: '',
  condition: '',
}

const toDraft = (c: FilterConditions): Draft => ({
  category: c.category ?? '',
  courseName: c.courseName ?? '',
  priceMin: c.priceMin?.toString() ?? '',
  priceMax: c.priceMax?.toString() ?? '',
  campus: c.campus ?? '',
  condition: c.condition ?? '',
})

const toNumber = (value: string) => (value === '' ? undefined : Number(value))

function toConditions(d: Draft): FilterConditions {
  const conditions = {
    category: d.category || undefined,
    courseName: d.courseName.trim() || undefined,
    priceMin: toNumber(d.priceMin),
    priceMax: toNumber(d.priceMax),
    campus: d.campus || undefined,
    condition: d.condition || undefined,
  } as FilterConditions
  return Object.fromEntries(
    Object.entries(conditions).filter(([, v]) => v !== undefined),
  ) as FilterConditions
}

type ChoiceProps<T extends string> = {
  legend: string
  /** 条件なしの選択肢の名前（すべて・指定なし） */
  anyLabel: string
  options: Option<T>[]
  value: string
  onChange: (value: string) => void
}

/** 1つだけ選ぶチップの並び。選び直すと前の選択は外れる */
function ChoiceChips<T extends string>({
  legend,
  anyLabel,
  options,
  value,
  onChange,
}: ChoiceProps<T>) {
  return (
    <fieldset className={styles.fieldset}>
      <legend className={styles.legend}>{legend}</legend>
      <div className={styles.chips}>
        <Chip label={anyLabel} selected={value === ''} onClick={() => onChange('')} />
        {options.map((o) => (
          <Chip
            key={o.value}
            label={o.label}
            selected={value === o.value}
            onClick={() => onChange(o.value)}
          />
        ))}
      </div>
    </fieldset>
  )
}

/** 価格の入力は数字だけ（出品できる価格は6桁まで。ITEM） */
const digits = (value: string) => value.replace(/\D/g, '').slice(0, 6)

/**
 * 検索の絞り込み（BROWSE、SearchFilter）。カテゴリ・授業名・価格帯・キャンパス・状態を選び、
 * 「この条件で検索」で確定する。「クリア」は選択を外すだけで、確定するまで検索結果は変えない。
 * 開いている間だけ描画する（閉じると入力中の内容は捨てる）。
 */
export function SearchFilterSheet({ conditions, onApply, onClose }: Props) {
  const [draft, setDraft] = useState(() => toDraft(conditions))
  const [priceError, setPriceError] = useState<string | null>(null)
  const priceErrorId = useId()

  const update = <K extends keyof Draft>(key: K, value: Draft[K]) => {
    setDraft((d) => ({ ...d, [key]: value }))
    if (key === 'priceMin' || key === 'priceMax') setPriceError(null)
  }

  const submit = (e: FormEvent) => {
    e.preventDefault()
    const next = toConditions(draft)
    if (
      next.priceMin !== undefined &&
      next.priceMax !== undefined &&
      next.priceMin > next.priceMax
    ) {
      setPriceError('上限は下限以上にしてください')
      return
    }
    onApply(next)
  }

  return (
    <BottomSheet
      open
      onClose={onClose}
      title="絞り込み"
      action={
        <Button
          variant="text"
          size="sm"
          className={styles.clear}
          onClick={() => {
            setDraft(emptyDraft)
            setPriceError(null)
          }}
        >
          クリア
        </Button>
      }
    >
      <form className={styles.form} onSubmit={submit} noValidate>
        <ChoiceChips
          legend="カテゴリ"
          anyLabel="すべて"
          options={categoryOptions}
          value={draft.category}
          onChange={(v) => update('category', v)}
        />
        <TextField
          label="授業名"
          size="md"
          placeholder="例：線形代数学I"
          maxLength={SEARCH_TEXT_MAX}
          value={draft.courseName}
          onChange={(e) => update('courseName', e.target.value)}
        />
        <fieldset className={styles.fieldset}>
          <legend className={styles.legend}>価格帯</legend>
          <div className={styles.priceRow}>
            <TextField
              label="価格の下限"
              hideLabel
              size="md"
              prefix="¥"
              numeric
              inputMode="numeric"
              placeholder="下限なし"
              className={styles.price}
              value={draft.priceMin}
              invalid={!!priceError}
              aria-describedby={priceError ? priceErrorId : undefined}
              onChange={(e) => update('priceMin', digits(e.target.value))}
            />
            <span className={styles.tilde} aria-hidden="true">
              〜
            </span>
            <TextField
              label="価格の上限"
              hideLabel
              size="md"
              prefix="¥"
              numeric
              inputMode="numeric"
              placeholder="上限なし"
              className={styles.price}
              value={draft.priceMax}
              invalid={!!priceError}
              aria-describedby={priceError ? priceErrorId : undefined}
              onChange={(e) => update('priceMax', digits(e.target.value))}
            />
          </div>
          {priceError && (
            <p id={priceErrorId} role="alert" className={styles.error}>
              {priceError}
            </p>
          )}
        </fieldset>
        <ChoiceChips
          legend="受け渡し希望キャンパス"
          anyLabel="指定なし"
          options={campusOptions}
          value={draft.campus}
          onChange={(v) => update('campus', v)}
        />
        <ChoiceChips
          legend="状態"
          anyLabel="指定なし"
          options={conditionOptions}
          value={draft.condition}
          onChange={(v) => update('condition', v)}
        />
        <div className={styles.footer}>
          <Button type="submit" fullWidth>
            この条件で検索
          </Button>
        </div>
      </form>
    </BottomSheet>
  )
}
