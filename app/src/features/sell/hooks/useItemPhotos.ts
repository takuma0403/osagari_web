import { useCallback, useEffect, useRef, useState } from 'react'
import { isApiError } from '../../../api/client'
import { isImageResizeError, resizeImage } from '../../../lib/imageResize'
import { uploadToSignedUrl } from '../../../lib/signedUpload'
import { createItemImageUpload, type OwnItem } from '../api'
import { ITEM_IMAGES_MAX } from '../form'

/** 出品画面に並べる写真。objectKey はアップロードが終わるまで undefined */
export type ItemPhoto = {
  /** 画面内で写真を区別するためのキー */
  id: string
  /** 表示する画像（アップロード前に縮小した画像の object URL か、閲覧用の署名付きURL） */
  url: string
  objectKey?: string
}

const unexpectedMessage = '写真を追加できませんでした。もう一度お試しください'

function messageOf(err: unknown): string {
  return isApiError(err) || isImageResizeError(err) ? err.message : unexpectedMessage
}

let seq = 0
const nextId = () => `photo-${++seq}`

/** 編集する商品の写真（閲覧用の署名付きURL）から並びを作る */
function photosFromItem(item: OwnItem | undefined): ItemPhoto[] {
  return (item?.images ?? []).map((image) => ({
    id: nextId(),
    url: image.url,
    objectKey: image.objectKey,
  }))
}

/**
 * 商品写真の並び（ITEM, STORAGE）。並び順が表示順で、最大10枚。
 * 追加は1枚ずつ「ブラウザで縮小 → 署名付きURLの発行 → 直接 PUT」を行い（学生証と同じ共通処理）、
 * 終わった写真から objectKey が入る。送るときは objectKey を並び順どおりに createItem・updateItem に渡す。
 * 編集では、読み込み終わった商品を渡して作る（並びは最初の1回だけ商品から作る）。
 */
export function useItemPhotos(item?: OwnItem) {
  const [photos, setPhotos] = useState<ItemPhoto[]>(() => photosFromItem(item))
  const [uploading, setUploading] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const objectUrls = useRef<string[]>([])

  // 画面を離れたら、縮小した画像の object URL を解放する
  useEffect(
    () => () => {
      objectUrls.current.forEach((url) => URL.revokeObjectURL(url))
    },
    [],
  )

  const add = useCallback(
    async (files: File[]) => {
      setError(null)
      const room = ITEM_IMAGES_MAX - photos.length
      if (files.length > room) setError('写真は10枚までです')
      for (const file of files.slice(0, Math.max(0, room))) {
        const id = nextId()
        setUploading((n) => n + 1)
        try {
          const { blob, contentType } = await resizeImage(file)
          const url = URL.createObjectURL(blob)
          objectUrls.current.push(url)
          setPhotos((list) => [...list, { id, url }])
          const upload = await createItemImageUpload(contentType)
          await uploadToSignedUrl(upload, blob)
          setPhotos((list) =>
            list.map((p) => (p.id === id ? { ...p, objectKey: upload.objectKey } : p)),
          )
        } catch (err) {
          setPhotos((list) => list.filter((p) => p.id !== id))
          setError(messageOf(err))
        } finally {
          setUploading((n) => n - 1)
        }
      }
    },
    [photos.length],
  )

  const remove = useCallback((id: string) => {
    setPhotos((list) => list.filter((p) => p.id !== id))
  }, [])

  /** 写真を offset だけ前（-1）か後ろ（+1）へ動かす */
  const move = useCallback((id: string, offset: -1 | 1) => {
    setPhotos((list) => {
      const from = list.findIndex((p) => p.id === id)
      const to = from + offset
      if (from < 0 || to < 0 || to >= list.length) return list
      const next = [...list]
      ;[next[from], next[to]] = [next[to], next[from]]
      return next
    })
  }, [])

  const objectKeys = photos.flatMap((p) => (p.objectKey ? [p.objectKey] : []))

  return { photos, objectKeys, uploading: uploading > 0, error, add, remove, move }
}
