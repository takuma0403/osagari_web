import type { BarcodeFormat } from '@zxing/library'

/** 読み取りを止める操作 */
export type ScanControls = { stop: () => void }

/** カメラの映像を video に映し、読み取ったコードの値を onText に渡し続ける */
export type StartScan = (
  video: HTMLVideoElement,
  onText: (text: string) => void,
) => Promise<ScanControls>

/** 読み取る形式を名前で指定する（@zxing/library を静的に読み込まないため、列挙値ではなく名前で受け取る） */
export type ScanFormat = keyof typeof BarcodeFormat

/**
 * 背面カメラで formats のコードだけを読み取る読み取り開始の関数を作る（FE）。
 * @zxing/browser は大きいので、読み取りを開いたときに読み込む。
 * カメラを使えないときは getUserMedia のエラー（NotAllowedError など）で失敗する。
 */
export function cameraScanner(formats: ScanFormat[]): StartScan {
  return async (video, onText) => {
    const [{ BrowserMultiFormatReader }, { BarcodeFormat, DecodeHintType }] = await Promise.all([
      import('@zxing/browser'),
      import('@zxing/library'),
    ])
    const hints = new Map([
      [DecodeHintType.POSSIBLE_FORMATS, formats.map((format) => BarcodeFormat[format])],
    ])
    const reader = new BrowserMultiFormatReader(hints)
    return reader.decodeFromConstraints(
      { video: { facingMode: 'environment' }, audio: false },
      video,
      (result) => {
        if (result) onText(result.getText())
      },
    )
  }
}
