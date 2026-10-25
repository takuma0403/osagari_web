import { useState } from 'react'
import { BottomSheet } from '../../../../components/BottomSheet/BottomSheet'
import { Button } from '../../../../components/Button/Button'
import { InfoBox } from '../../../../components/InfoBox/InfoBox'
import { isBookBarcode } from '../../../../lib/isbn'
import { isCameraDenied, useCameraScan } from '../../../../lib/useCameraScan'
import { startIsbnScan, type StartScan } from './startIsbnScan'
import styles from './IsbnScanner.module.css'

type Props = {
  open: boolean
  onClose: () => void
  /** 書籍のバーコード（978・979 で始まる ISBN-13）を読み取ったとき */
  onDetect: (isbn: string) => void
  /** 読み取りの開始（テスト・Storybook でカメラの代わりを渡す） */
  startScan?: StartScan
}

function cameraErrorMessage(err: unknown): string {
  if (isCameraDenied(err))
    return 'カメラの使用が許可されていません。ブラウザの設定で許可するか、ISBN を入力してください'
  return 'カメラを起動できませんでした。ISBN を入力してください'
}

/**
 * ISBN のバーコードをカメラで読み取るシート（ITEM）。開いている間だけカメラを使い、閉じたら止める。
 * 本の裏表紙には2段のバーコードがあり、上段（978・979 で始まる）だけが ISBN なので、それ以外は読み流す。
 */
export function IsbnScanner({ open, onClose, onDetect, startScan = startIsbnScan }: Props) {
  const [wrongCode, setWrongCode] = useState(false)
  const { videoRef, error } = useCameraScan({
    active: open,
    startScan,
    onText: (text) => {
      if (!isBookBarcode(text)) {
        setWrongCode(true)
        return false
      }
      onDetect(text)
      return true
    },
  })
  // 開き直したら、前回の「ISBN ではない」の表示を消す
  const [wasOpen, setWasOpen] = useState(open)
  if (open !== wasOpen) {
    setWasOpen(open)
    if (open) setWrongCode(false)
  }

  return (
    <BottomSheet
      open={open}
      onClose={onClose}
      title="バーコードを読み取る"
      description="本の裏表紙にある、978 か 979 で始まるバーコードを枠に入れてください"
    >
      <div className={styles.camera}>
        <video
          ref={videoRef}
          className={styles.video}
          muted
          playsInline
          aria-label="カメラの映像"
        />
      </div>
      {error ? (
        <InfoBox tone="danger" icon="alert" role="alert">
          {cameraErrorMessage(error)}
        </InfoBox>
      ) : (
        wrongCode && (
          <p role="status" className={styles.hint}>
            ISBN ではないバーコードです。978 か 979 で始まる上の段を写してください
          </p>
        )
      )}
      <Button variant="ghost" size="md" fullWidth onClick={onClose}>
        閉じる
      </Button>
    </BottomSheet>
  )
}
