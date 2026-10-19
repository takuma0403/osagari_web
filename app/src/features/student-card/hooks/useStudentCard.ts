import { useCallback, useEffect, useState } from 'react'
import { isApiError } from '../../../api/client'
import { isImageResizeError, resizeImage } from '../../../lib/imageResize'
import { uploadToSignedUrl } from '../../../lib/signedUpload'
import { useSession } from '../../auth/session'
import {
  createStudentCardUpload,
  getCurrentStudentCard,
  submitStudentCard,
  type StudentCardState,
} from '../api'

const unexpectedMessage = '学生証を登録できませんでした。もう一度お試しください'

function messageOf(err: unknown): string {
  return isApiError(err) || isImageResizeError(err) ? err.message : unexpectedMessage
}

/**
 * 本人の学生証の審査状況と、画像の登録（CARD, STORAGE）。
 * 登録は「ブラウザで縮小 → 署名付きURLの発行 → 直接 PUT → オブジェクト名を登録」の順に行う。
 * URL は一度しか使えないため、撮り直しのたびに発行し直す。
 * 本人には画像の閲覧URLが返らないので、登録した画像はこの画面で縮小した画像のプレビューだけを出す（再読み込みすると消える）。
 */
export function useStudentCard() {
  const { user, signIn } = useSession()
  const [card, setCard] = useState<StudentCardState>()
  const [loadError, setLoadError] = useState<string | null>(null)
  const [reloadKey, setReloadKey] = useState(0)
  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [preview, setPreview] = useState<string | null>(null)

  useEffect(() => {
    const controller = new AbortController()
    getCurrentStudentCard(controller.signal)
      .then(setCard)
      .catch((err: unknown) => {
        if (!controller.signal.aborted) setLoadError(messageOf(err))
      })
    return () => controller.abort()
  }, [reloadKey])

  // 差し替えたときと画面を離れたときに、プレビューの URL を解放する
  useEffect(
    () => () => {
      if (preview) URL.revokeObjectURL(preview)
    },
    [preview],
  )

  const reload = useCallback(() => {
    setLoadError(null)
    setReloadKey((key) => key + 1)
  }, [])

  const submit = useCallback(
    async (file: File) => {
      setSubmitting(true)
      setSubmitError(null)
      try {
        const { blob, contentType } = await resizeImage(file)
        const upload = await createStudentCardUpload(contentType)
        await uploadToSignedUrl(upload, blob)
        const next = await submitStudentCard(upload.objectKey)
        setCard(next)
        setPreview(URL.createObjectURL(blob))
        // マイページなどの表示と揃える
        if (user) signIn({ ...user, sellerStatus: next.status })
      } catch (err) {
        setSubmitError(messageOf(err))
        // 承認済みになっていた（409）ときは、最新の審査状況を取り直す
        if (isApiError(err) && err.code === 'invalid_status_transition') reload()
      } finally {
        setSubmitting(false)
      }
    },
    [user, signIn, reload],
  )

  return { card, loadError, reload, submit, submitting, submitError, preview }
}
