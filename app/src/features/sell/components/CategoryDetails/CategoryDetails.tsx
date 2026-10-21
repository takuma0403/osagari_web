import { useState, type ReactNode } from 'react'
import { Icon } from '../../../../components/Icon/Icon'
import { TextField } from '../../../../components/TextField/TextField'
import { ITEM_DETAIL_MAX, type SellErrors, type SellForm } from '../../form'
import { IsbnScanner } from '../IsbnScanner/IsbnScanner'
import type { StartScan } from '../IsbnScanner/startIsbnScan'
import styles from './CategoryDetails.module.css'

type Props = {
  form: SellForm
  onChange: <K extends keyof SellForm>(key: K, value: SellForm[K]) => void
  errors: SellErrors
  /** ISBN の読み取り（テスト・Storybook でカメラの代わりを渡す） */
  startScan?: StartScan
}

/**
 * カテゴリ別の追加項目（ITEM）。教科書・参考書は授業名・担当教員・ISBN・書き込みの有無、家具・家電はサイズ・動作確認の有無。
 * ゲーム・その他と未選択では何も出さない。どの項目も任意。
 */
export function CategoryDetails({ form, onChange, errors, startScan }: Props) {
  const [scanning, setScanning] = useState(false)

  if (form.category === 'textbook') {
    return (
      <Box legend="教科書の情報">
        <DetailField
          label="授業名"
          value={form.courseName}
          error={errors.courseName}
          onChange={(v) => onChange('courseName', v)}
        />
        <DetailField
          label="担当教員"
          value={form.instructor}
          error={errors.instructor}
          onChange={(v) => onChange('instructor', v)}
        />
        <div className={styles.isbn}>
          <DetailField
            label="ISBN"
            value={form.isbn}
            error={errors.isbn}
            onChange={(v) => onChange('isbn', v)}
            inputMode="numeric"
            numeric
            className={styles.isbnField}
          />
          <button
            type="button"
            className={styles.scan}
            aria-label="バーコードを読み取る"
            onClick={() => setScanning(true)}
          >
            <Icon name="barcode" size={20} />
          </button>
        </div>
        <Check
          label="書き込みあり"
          checked={form.hasWriting}
          onChange={(v) => onChange('hasWriting', v)}
        />
        <IsbnScanner
          open={scanning}
          onClose={() => setScanning(false)}
          onDetect={(isbn) => {
            onChange('isbn', isbn)
            setScanning(false)
          }}
          startScan={startScan}
        />
      </Box>
    )
  }
  if (form.category === 'furniture_appliance') {
    return (
      <Box legend="家具・家電の情報">
        <DetailField
          label="サイズ"
          value={form.size}
          error={errors.size}
          onChange={(v) => onChange('size', v)}
        />
        <Check
          label="動作確認済み"
          checked={form.operationChecked}
          onChange={(v) => onChange('operationChecked', v)}
        />
      </Box>
    )
  }
  return null
}

function Box({ legend, children }: { legend: string; children: ReactNode }) {
  return (
    <fieldset className={styles.box}>
      <legend className={styles.legend}>{legend}</legend>
      {children}
    </fieldset>
  )
}

type DetailFieldProps = {
  label: string
  value: string
  error?: string
  onChange: (value: string) => void
  inputMode?: 'numeric'
  numeric?: boolean
  className?: string
}

/** 追加項目の1行入力。ラベルはプレースホルダで示す（デザインどおり。読み上げにはラベルを残す） */
function DetailField({
  label,
  value,
  error,
  onChange,
  inputMode,
  numeric,
  className,
}: DetailFieldProps) {
  return (
    <TextField
      label={label}
      hideLabel
      placeholder={label}
      size="sm"
      tone="primary"
      value={value}
      error={error}
      onChange={(e) => onChange(e.target.value)}
      inputMode={inputMode}
      numeric={numeric}
      maxLength={numeric ? undefined : ITEM_DETAIL_MAX}
      className={className}
    />
  )
}

function Check({
  label,
  checked,
  onChange,
}: {
  label: string
  checked: boolean
  onChange: (v: boolean) => void
}) {
  return (
    <label className={styles.check}>
      {label}
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} />
    </label>
  )
}
