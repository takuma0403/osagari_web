/** アップロード前に縮小するときの長辺の上限（STORAGE） */
export const IMAGE_MAX_LONG_SIDE = 1600
/** 署名付きURLが受け付ける大きさの上限（STORAGE。サーバーの x-goog-content-length-range と同じ） */
export const IMAGE_MAX_BYTES = 5 * 1024 * 1024

/** 署名付きURLで送れる画像の形式（STORAGE） */
export type UploadContentType = 'image/jpeg' | 'image/webp'

export type ResizedImage = {
  blob: Blob
  contentType: UploadContentType
}

/** 画像を縮小できなかったことを表す。message はそのまま利用者に表示できる */
export class ImageResizeError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'ImageResizeError'
  }
}

export function isImageResizeError(err: unknown): err is ImageResizeError {
  return err instanceof ImageResizeError
}

/** 縦横比を保って長辺を maxLongSide 以下にしたときの大きさ。小さい画像は拡大しない */
export function fitWithin(
  width: number,
  height: number,
  maxLongSide: number = IMAGE_MAX_LONG_SIDE,
): { width: number; height: number } {
  const scale = Math.min(1, maxLongSide / Math.max(width, height))
  return {
    width: Math.max(1, Math.round(width * scale)),
    height: Math.max(1, Math.round(height * scale)),
  }
}

// 上限を超えたときに順に試す圧縮の品質
const QUALITIES = [0.85, 0.7, 0.55, 0.4]

function toBlob(canvas: HTMLCanvasElement, type: string, quality: number): Promise<Blob | null> {
  return new Promise((resolve) => canvas.toBlob(resolve, type, quality))
}

type Options = {
  maxLongSide?: number
  maxBytes?: number
}

/**
 * 画像をブラウザで縮小・圧縮する（STORAGE）。長辺を1600px以下にして WebP で書き出し、
 * WebP で書き出せないブラウザでは JPEG にする。maxBytes を超えたら品質を下げて圧縮し直す。
 * 学生証の登録と出品の写真で共通に使う。
 */
export async function resizeImage(
  file: Blob,
  { maxLongSide = IMAGE_MAX_LONG_SIDE, maxBytes = IMAGE_MAX_BYTES }: Options = {},
): Promise<ResizedImage> {
  let bitmap: ImageBitmap
  try {
    // 写真の向き（EXIF）を反映して読み込む
    bitmap = await createImageBitmap(file, { imageOrientation: 'from-image' })
  } catch {
    throw new ImageResizeError(
      'この画像は読み込めませんでした。JPEG・PNG・WebP の画像を選んでください',
    )
  }

  try {
    const { width, height } = fitWithin(bitmap.width, bitmap.height, maxLongSide)
    const canvas = document.createElement('canvas')
    canvas.width = width
    canvas.height = height
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new ImageResizeError('画像を縮小できませんでした')
    // JPEG は透過できないため、透過部分が黒くならないよう白で塗ってから描く
    ctx.fillStyle = '#fff'
    ctx.fillRect(0, 0, width, height)
    ctx.drawImage(bitmap, 0, 0, width, height)

    let contentType: UploadContentType = 'image/webp'
    for (const quality of QUALITIES) {
      let blob = await toBlob(canvas, contentType, quality)
      // WebP に対応していないブラウザは PNG で返すので、JPEG に切り替える
      if (blob?.type !== contentType && contentType === 'image/webp') {
        contentType = 'image/jpeg'
        blob = await toBlob(canvas, contentType, quality)
      }
      if (!blob) throw new ImageResizeError('画像を縮小できませんでした')
      if (blob.size <= maxBytes) return { blob, contentType }
    }
    throw new ImageResizeError('画像が大きすぎます。別の画像を選んでください')
  } finally {
    bitmap.close()
  }
}
