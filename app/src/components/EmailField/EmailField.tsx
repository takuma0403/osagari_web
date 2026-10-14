import { normalizeStudentId, UNIVERSITY_EMAIL_DOMAIN } from '../../lib/studentId'
import { TextField, type TextFieldProps } from '../TextField/TextField'
import styles from './EmailField.module.css'

type Props = Omit<TextFieldProps, 'label' | 'value' | 'onChange' | 'suffix' | 'type'> & {
  label?: string
  /** 学籍番号（@ より前の8桁） */
  value: string
  onChange: (studentId: string) => void
}

/**
 * 大学のメールアドレスの入力（DS-3）。学籍番号8桁の入力欄に、固定のドメインを続けて表示する。
 * 数字以外は受け付けない。登録できるかどうかの最終的な検証はサーバーが行う（AUTHN-2）。
 */
export function EmailField({ label = '大学のメールアドレス', value, onChange, ...rest }: Props) {
  return (
    <TextField
      {...rest}
      label={label}
      value={value}
      onChange={(e) => onChange(normalizeStudentId(e.target.value))}
      suffix={UNIVERSITY_EMAIL_DOMAIN}
      numeric
      inputMode="numeric"
      autoComplete="username"
      // アドレスごと貼り付けられても @ より前を取り出せるよう、長さは normalizeStudentId で切る
      inputClassName={styles.studentId}
    />
  )
}
