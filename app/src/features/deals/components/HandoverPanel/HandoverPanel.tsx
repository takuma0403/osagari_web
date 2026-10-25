import { Button } from '../../../../components/Button/Button'
import { QrCode } from '../../../../components/QrCode/QrCode'
import { formatCountdown, formatHandoverCode } from '../../../../lib/format'
import { useCountdown } from '../../../auth/hooks/useCountdown'
import styles from './HandoverPanel.module.css'

type Props = {
  /** QR に載せる URL（`/deals/{取引ID}/complete#q=…`）。発行中は null */
  url: string | null
  /** QR の下に出す6桁の確認コード */
  code: string | null
  /** 有効期限（ISO 8601。発行から5分） */
  expiresAt: string | null
  /** 「表示し直す」（発行し直す） */
  onReissue: () => void
  /** 発行し直している間 */
  reissuing?: boolean
}

/**
 * 受け渡しQR（DEAL-3、DealQR）。出品者が表示し、購入者に読み取ってもらう。
 * QR の中身は取引IDと使い捨てのトークンだけ。カメラが使えないときのために6桁の確認コードも出す。
 * 期限（5分）が切れたら QR を薄くして読み取らせず、「表示し直す」で発行し直してもらう。
 */
export function HandoverPanel({ url, code, expiresAt, onReissue, reissuing }: Props) {
  const remaining = useCountdown(expiresAt ? new Date(expiresAt).getTime() : 0)
  const expired = expiresAt !== null && remaining <= 0
  return (
    <section className={styles.panel}>
      <p className={styles.lead}>
        商品を確認してもらってから、
        <br />
        購入者にこのコードを読み取ってもらいます。
      </p>
      <div className={styles.frame}>
        {url ? (
          <QrCode value={url} label="受け渡し用QRコード" dimmed={expired} />
        ) : (
          <div className={styles.placeholder} role="status">
            QRコードを用意しています…
          </div>
        )}
      </div>
      {code && !expired && (
        <div className={styles.code}>
          <span className={styles.codeLabel}>カメラが使えないときの確認コード</span>
          <span className={styles.codeValue}>{formatHandoverCode(code)}</span>
        </div>
      )}
      <div className={styles.expiry}>
        {expired ? (
          <span role="status">有効期限が切れました</span>
        ) : (
          expiresAt && (
            <span>
              あと <b className={styles.remaining}>{formatCountdown(remaining)}</b> で無効
            </span>
          )
        )}
        <Button
          variant={expired ? 'primary' : 'ghost'}
          size="sm"
          onClick={onReissue}
          disabled={reissuing}
        >
          表示し直す
        </Button>
      </div>
    </section>
  )
}
