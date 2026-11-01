import { useCallback, useEffect, useState } from 'react'
import { isApiError } from '../../../api/client'
import { createItem, getFeeRate, getOwnItem, publishItem, updateItem, type OwnItem } from '../api'
import {
  emptySellForm,
  formFromItem,
  toItemInput,
  validateSellForm,
  type SellErrors,
  type SellForm,
} from '../form'

const unexpectedMessage = '保存できませんでした。もう一度お試しください'
const loadFailedMessage = '商品を読み込めませんでした'

/** 編集できるステータス（ITEM, ADR 0016）。購入希望あり・取引中・取引完了・停止は編集できない */
const editableStatuses: OwnItem['status'][] = ['draft', 'published', 'withdrawn']

/** 送信の種類。save は下書き・取り下げ・公開中のまま保存、publish は公開（再出品を含む） */
export type SellAction = 'save' | 'publish'

/**
 * 出品・編集の画面の状態（ITEM, PAY-2）。itemId があれば編集で、自分の商品と手数料率を読み込む。
 * 送信は、新規なら createItem（下書きか公開）、編集なら updateItem の後、公開するときだけ publishItem を呼ぶ。
 * 成功したら保存後の商品を返す（失敗したら undefined）。
 */
export function useSellForm(itemId: string | undefined) {
  const [item, setItem] = useState<OwnItem>()
  const [loadError, setLoadError] = useState<string | null>(null)
  const [reloadKey, setReloadKey] = useState(0)
  const [feeRate, setFeeRate] = useState<number>()
  const [form, setForm] = useState<SellForm>(emptySellForm)
  const [errors, setErrors] = useState<SellErrors>({})
  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)

  useEffect(() => {
    if (!itemId) return
    const controller = new AbortController()
    getOwnItem(itemId, controller.signal)
      .then((loaded) => {
        setItem(loaded)
        setForm(formFromItem(loaded))
      })
      .catch((err: unknown) => {
        if (controller.signal.aborted) return
        setLoadError(isApiError(err) && err.code !== 'not_found' ? err.message : loadFailedMessage)
      })
    return () => controller.abort()
  }, [itemId, reloadKey])

  // 料率が取れなかったときは見込みの表示を出さないだけにする（出品はできる）
  useEffect(() => {
    const controller = new AbortController()
    getFeeRate(controller.signal)
      .then((rate) => setFeeRate(rate.feeRateBasisPoints))
      .catch(() => {})
    return () => controller.abort()
  }, [])

  const reload = useCallback(() => {
    setLoadError(null)
    setReloadKey((key) => key + 1)
  }, [])

  const update = useCallback(<K extends keyof SellForm>(key: K, value: SellForm[K]) => {
    setForm((f) => ({ ...f, [key]: value }))
    setErrors((e) => ({ ...e, [key]: undefined }))
  }, [])

  const clearError = useCallback((key: keyof SellErrors) => {
    setErrors((e) => ({ ...e, [key]: undefined }))
  }, [])

  const editable = !item || editableStatuses.includes(item.status)

  const submit = useCallback(
    async (action: SellAction, imageObjectKeys: string[]): Promise<OwnItem | undefined> => {
      setSubmitError(null)
      // 公開中の商品は、公開のままの保存でもすべての共通項目が必要（ADR 0016）
      const full = action === 'publish' || item?.status === 'published'
      const found = validateSellForm(form, imageObjectKeys.length, full)
      setErrors(found)
      if (Object.keys(found).length > 0) return undefined

      setSubmitting(true)
      try {
        const input = toItemInput(form, imageObjectKeys)
        let saved: OwnItem
        if (!item) {
          saved = await createItem(input, action === 'publish')
        } else {
          saved = await updateItem(item.id, input)
          if (action === 'publish' && saved.status !== 'published')
            saved = await publishItem(item.id)
        }
        setItem(saved)
        return saved
      } catch (err) {
        setSubmitError(isApiError(err) ? err.message : unexpectedMessage)
        return undefined
      } finally {
        setSubmitting(false)
      }
    },
    [form, item],
  )

  return {
    item,
    loading: Boolean(itemId) && !item && !loadError,
    loadError,
    reload,
    editable,
    feeRate,
    form,
    update,
    errors,
    clearError,
    submit,
    submitting,
    submitError,
  }
}
