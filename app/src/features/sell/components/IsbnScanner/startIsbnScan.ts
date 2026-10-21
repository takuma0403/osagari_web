/** 読み取りを止める操作 */
export type ScanControls = { stop: () => void }

/** カメラの映像を video に映し、読み取ったバーコードの値を onText に渡し続ける */
export type StartScan = (
  video: HTMLVideoElement,
  onText: (text: string) => void,
) => Promise<ScanControls>

/**
 * 背面カメラで EAN-13（書籍の ISBN バーコード）を読み取る（ITEM）。
 * @zxing/browser は大きいので、読み取りを開いたときに読み込む。
 * カメラを使えないときは getUserMedia のエラー（NotAllowedError など）で失敗する。
 */
export const startIsbnScan: StartScan = async (video, onText) => {
  const [{ BrowserMultiFormatReader }, { BarcodeFormat, DecodeHintType }] = await Promise.all([
    import('@zxing/browser'),
    import('@zxing/library'),
  ])
  const hints = new Map([[DecodeHintType.POSSIBLE_FORMATS, [BarcodeFormat.EAN_13]]])
  const reader = new BrowserMultiFormatReader(hints)
  return reader.decodeFromConstraints(
    { video: { facingMode: 'environment' }, audio: false },
    video,
    (result) => {
      if (result) onText(result.getText())
    },
  )
}
