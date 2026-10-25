import { useMemo } from 'react'
import { encode } from 'uqr'
import styles from './QrCode.module.css'

type Props = {
  /** QR に載せる文字列 */
  value: string
  /** 読み上げる名前（例 受け渡し用QRコード） */
  label: string
  /** 期限切れなど、読み取らせたくないときに薄くする */
  dimmed?: boolean
}

/** 黒のマスを1本の path にする（1マス＝1単位の正方形） */
function pathOf(data: boolean[][]): string {
  const parts: string[] = []
  data.forEach((row, y) =>
    row.forEach((on, x) => {
      if (on) parts.push(`M${x} ${y}h1v1h-1z`)
    }),
  )
  return parts.join('')
}

/** QR コード（DEAL-3）。uqr で作った行列を SVG で描く。周りの余白（クワイエットゾーン）は2マス */
export function QrCode({ value, label, dimmed }: Props) {
  const { size, d } = useMemo(() => {
    const qr = encode(value, { ecc: 'M', border: 2 })
    return { size: qr.size, d: pathOf(qr.data) }
  }, [value])
  return (
    <svg
      role="img"
      aria-label={label}
      viewBox={`0 0 ${size} ${size}`}
      shapeRendering="crispEdges"
      className={[styles.qr, dimmed && styles.dimmed].filter(Boolean).join(' ')}
    >
      <rect width={size} height={size} className={styles.light} />
      <path d={d} className={styles.dark} />
    </svg>
  )
}
