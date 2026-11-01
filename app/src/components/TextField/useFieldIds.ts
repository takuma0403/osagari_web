import { useId } from 'react'
import type { FieldProps } from './Field'

/** 入力欄の id と、補足・エラーとの関連付け（aria-describedby）を決める */
export function useFieldIds(
  id: string | undefined,
  { hint, error }: Pick<FieldProps, 'hint' | 'error'>,
) {
  const autoId = useId()
  const inputId = id ?? autoId
  const hintId = `${inputId}-hint`
  const errorId = `${inputId}-error`
  const describedBy = [hint && hintId, error && errorId].filter(Boolean).join(' ') || undefined
  return { inputId, hintId, errorId, describedBy }
}
