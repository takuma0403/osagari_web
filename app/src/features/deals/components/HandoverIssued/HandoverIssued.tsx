import { useId, useState } from 'react'
import { Button } from '../../../../components/Button/Button'
import { Icon } from '../../../../components/Icon/Icon'
import { InfoBox } from '../../../../components/InfoBox/InfoBox'
import { formatCountdownHours, formatHandoverCode } from '../../../../lib/format'
import { copyText } from '../../../../lib/share'
import { useCountdown } from '../../../auth/hooks/useCountdown'
import styles from './HandoverIssued.module.css'

type Props = {
  /** link：完了リンク／code：6桁の完了コード */
  kind: 'link' | 'code'
  /** 完了リンクの URL か、6桁の完了コード */
  value: string
  /** 有効期限（ISO 8601。発行から24時間） */
  expiresAt: string
}

/**
 * 発行した完了リンク・完了コード（DEAL-3、DealCompleteIssue）。QRを読み取ってもらい忘れたときに、出品者が Teams で購入者に送る。
 * 完了させるのは購入者の操作で、使えるのは1回だけ。
 */
export function HandoverIssued({ kind, value, expiresAt }: Props) {
  const inputId = useId()
  const remaining = useCountdown(new Date(expiresAt).getTime())
  const [copied, setCopied] = useState(false)
  const [copyError, setCopyError] = useState(false)
  const isLink = kind === 'link'

  const copy = () => {
    setCopyError(false)
    copyText(value).then(
      () => setCopied(true),
      () => setCopyError(true),
    )
  }
  const copyLabel = copied ? 'コピーしました' : 'コピー'

  return (
    <>
      <section className={styles.card}>
        <div className={styles.heading}>
          <span className={styles.check}>
            <Icon name="check" size={26} strokeWidth={2} />
          </span>
          <h2 className={styles.title}>
            {isLink ? '完了リンクを発行しました' : '完了コードを発行しました'}
          </h2>
          <p className={styles.lead}>
            Teamsで購入者に送ってください。購入者が開く（入力する）と受け渡し完了になります。
          </p>
        </div>
        {isLink ? (
          <div className={styles.link}>
            <label htmlFor={inputId} className={styles.label}>
              完了リンク
            </label>
            <div className={styles.linkRow}>
              <input
                id={inputId}
                readOnly
                value={value}
                className={styles.linkInput}
                onFocus={(e) => e.currentTarget.select()}
              />
              <Button size="md" onClick={copy}>
                {copyLabel}
              </Button>
            </div>
          </div>
        ) : (
          <div className={styles.code}>
            <span className={styles.codeLabel}>6桁の完了コード</span>
            <span className={styles.codeValue}>{formatHandoverCode(value)}</span>
            <Button size="sm" onClick={copy}>
              {copyLabel}
            </Button>
          </div>
        )}
        {copyError && (
          <p role="alert" className={styles.error}>
            コピーできませんでした。{isLink ? 'リンク' : 'コード'}を書き写してください
          </p>
        )}
        <dl className={styles.facts}>
          <dt>有効期限</dt>
          <dd>
            あと <b className={styles.num}>{formatCountdownHours(remaining)}</b>（24時間）
          </dd>
          <dt>使える回数</dt>
          <dd>1回（使うと無効になります）</dd>
        </dl>
      </section>
      <InfoBox icon="info">
        完了させるのは購入者です。購入者には「商品を受け取りましたか？」の確認が表示され、受け取ったあとに完了すると代金が確定します。
      </InfoBox>
    </>
  )
}
