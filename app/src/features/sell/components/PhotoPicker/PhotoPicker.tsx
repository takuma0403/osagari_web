import { useRef, useState, type ChangeEvent } from 'react'
import { BottomSheet } from '../../../../components/BottomSheet/BottomSheet'
import { Button } from '../../../../components/Button/Button'
import { Icon } from '../../../../components/Icon/Icon'
import { ITEM_IMAGES_MAX } from '../../form'
import type { ItemPhoto } from '../../hooks/useItemPhotos'
import styles from './PhotoPicker.module.css'

type Props = {
  photos: ItemPhoto[]
  /** 写真を選んだとき（複数可） */
  onAdd: (files: File[]) => void
  onRemove: (id: string) => void
  /** 写真を前（-1）か後ろ（+1）へ動かす */
  onMove: (id: string, offset: -1 | 1) => void
  /** アップロードのエラーや「写真を1枚以上追加してください」 */
  error?: string | null
  disabled?: boolean
}

/**
 * 商品写真の並び（ITEM, STORAGE）。80px のサムネイル＋「写真を追加」。並び順が表示順で、最大10枚。
 * 写真を押すとシートで並べ替え・削除を選べる（小さいボタンを写真に重ねず、44px のタップ領域を保つ）。
 */
export function PhotoPicker({ photos, onAdd, onRemove, onMove, error, disabled }: Props) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const selectedIndex = photos.findIndex((p) => p.id === selectedId)
  const selected = selectedIndex >= 0 ? photos[selectedIndex] : undefined
  const full = photos.length >= ITEM_IMAGES_MAX

  function handleChange(event: ChangeEvent<HTMLInputElement>) {
    const files = [...(event.target.files ?? [])]
    // 同じ画像を選び直しても change が起きるように空にする
    event.target.value = ''
    if (files.length > 0) onAdd(files)
  }

  function act(fn: () => void) {
    fn()
    setSelectedId(null)
  }

  return (
    <section className={styles.picker} aria-label="写真">
      <div className={styles.row}>
        {photos.map((photo, i) => {
          const uploading = !photo.objectKey
          const name = `写真${i + 1}`
          // 表紙のバッジはボタンの外に重ねる（ボタンの名前と見えている文字を食い違わせない）
          return (
            <div key={photo.id} className={styles.slot}>
              <button
                type="button"
                className={styles.thumb}
                aria-label={uploading ? `${name}（アップロード中）` : `${name}を並べ替え・削除`}
                aria-busy={uploading || undefined}
                disabled={disabled || uploading}
                onClick={() => setSelectedId(photo.id)}
              >
                <img src={photo.url} alt="" className={styles.image} />
                {uploading && <span className={styles.uploading} aria-hidden="true" />}
              </button>
              {i === 0 && <span className={styles.cover}>表紙</span>}
            </div>
          )
        })}
        {!full && (
          <button
            type="button"
            className={styles.add}
            disabled={disabled}
            onClick={() => inputRef.current?.click()}
          >
            <Icon name="camera" size={22} />
            写真を追加
          </button>
        )}
      </div>
      <input
        ref={inputRef}
        type="file"
        accept="image/*"
        multiple
        aria-label="商品の写真"
        className="visually-hidden"
        tabIndex={-1}
        onChange={handleChange}
      />
      <p className={styles.note}>
        {ITEM_IMAGES_MAX}枚まで（{photos.length}/{ITEM_IMAGES_MAX}）。1枚目が表紙になります
      </p>
      {error && (
        <p role="alert" className={styles.error}>
          {error}
        </p>
      )}
      <BottomSheet
        open={Boolean(selected)}
        onClose={() => setSelectedId(null)}
        title={`写真${selectedIndex + 1}`}
      >
        {selected && (
          <div className={styles.actions}>
            <Button
              variant="ghost"
              size="md"
              fullWidth
              disabled={selectedIndex === 0}
              onClick={() => act(() => onMove(selected.id, -1))}
            >
              前へ移動
            </Button>
            <Button
              variant="ghost"
              size="md"
              fullWidth
              disabled={selectedIndex === photos.length - 1}
              onClick={() => act(() => onMove(selected.id, 1))}
            >
              後ろへ移動
            </Button>
            <Button
              variant="danger"
              size="md"
              fullWidth
              onClick={() => act(() => onRemove(selected.id))}
            >
              削除する
            </Button>
            <Button variant="text" size="sm" fullWidth onClick={() => setSelectedId(null)}>
              閉じる
            </Button>
          </div>
        )}
      </BottomSheet>
    </section>
  )
}
