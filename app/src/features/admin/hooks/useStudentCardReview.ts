import { useCallback, useEffect, useRef, useState } from 'react'
import { isApiError } from '../../../api/client'
import { useSession } from '../../auth/session'
import {
  approveStudentCard,
  createStudentCardView,
  listPendingStudentCards,
  listStudentCardViews,
  rejectStudentCard,
  type AdminStudentCard,
  type StudentCardView,
} from '../api'

export type ReviewDecision = 'approve' | 'reject'

/** 選択中の学生証の画像。閲覧URLは選ぶたびに発行する（発行すると閲覧が記録される。STORAGE） */
export type CardImageState =
  { status: 'loading' } | { status: 'ready'; url: string } | { status: 'error'; message: string }

const unexpectedMessage = '読み込めませんでした。もう一度お試しください'

function messageOf(err: unknown): string {
  return isApiError(err) ? err.message : unexpectedMessage
}

/** 操作の結果のトースト。done は済んだこと、alert は状態が変わっていて一覧を取り直したこと */
export type ReviewNotice = { message: string; tone: 'done' | 'alert' }

const doneMessages: Record<ReviewDecision, string> = {
  approve: '承認しました',
  reject: '差し戻しました',
}

/**
 * 学生証の審査（ADMIN, CARD）。審査待ちを古い順に読み込み、先頭を選ぶ。
 * 選ぶたびに画像の閲覧URLを発行し（閲覧が記録される）、そのあと閲覧記録を取り直す。
 * 承認・差し戻しが済んだら一覧から外して次の1件を選ぶ。状態が変わっていた（409）・見つからない（404）ときは一覧を取り直す。
 */
export function useStudentCardReview() {
  const { user, signIn } = useSession()
  const [cards, setCards] = useState<AdminStudentCard[]>()
  const [loadError, setLoadError] = useState<string | null>(null)
  const [reloadKey, setReloadKey] = useState(0)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [image, setImage] = useState<CardImageState>({ status: 'loading' })
  const [views, setViews] = useState<StudentCardView[]>()
  const [viewsError, setViewsError] = useState<string | null>(null)
  const [deciding, setDeciding] = useState<ReviewDecision | null>(null)
  const [decisionError, setDecisionError] = useState<string | null>(null)
  const [notice, setNotice] = useState<ReviewNotice>()
  // 選び直したら前の学生証の読み込みを止める。発行は副作用（閲覧の記録）なので、effect ではなく選んだときに呼ぶ
  const selectController = useRef<AbortController | null>(null)

  const select = useCallback((id: string | null) => {
    selectController.current?.abort()
    setSelectedId(id)
    setImage({ status: 'loading' })
    setViews(undefined)
    setViewsError(null)
    setDecisionError(null)
    if (id === null) return
    const controller = new AbortController()
    selectController.current = controller
    const { signal } = controller
    createStudentCardView(id, signal)
      .then((img) => setImage({ status: 'ready', url: img.url }))
      .catch((err: unknown) => {
        if (!signal.aborted) setImage({ status: 'error', message: messageOf(err) })
      })
      // 発行した閲覧も含めるため、発行が終わってから閲覧記録を取る
      .then(() => (signal.aborted ? undefined : listStudentCardViews(id, signal)))
      .then((list) => {
        if (list) setViews(list)
      })
      .catch((err: unknown) => {
        if (!signal.aborted) setViewsError(messageOf(err))
      })
  }, [])

  useEffect(() => {
    const controller = new AbortController()
    listPendingStudentCards(controller.signal)
      .then((list) => {
        setCards(list)
        select(list[0]?.id ?? null)
      })
      .catch((err: unknown) => {
        if (!controller.signal.aborted) setLoadError(messageOf(err))
      })
    return () => controller.abort()
  }, [reloadKey, select])

  // 画面を離れたら、選択中の学生証の読み込みを止める
  useEffect(() => () => selectController.current?.abort(), [])

  // 操作の結果のトーストは少しだけ出して消す
  useEffect(() => {
    if (!notice) return
    const timer = setTimeout(() => setNotice(undefined), 3000)
    return () => clearTimeout(timer)
  }, [notice])

  const reload = useCallback(() => {
    setLoadError(null)
    setCards(undefined)
    setReloadKey((key) => key + 1)
  }, [])

  const decide = useCallback(
    async (decision: ReviewDecision) => {
      if (!cards || selectedId === null) return
      setDeciding(decision)
      setDecisionError(null)
      try {
        const reviewed = await (decision === 'approve' ? approveStudentCard : rejectStudentCard)(
          selectedId,
        )
        // 管理者が自分の学生証を審査したときは、マイページなどの表示と揃える（ADR 0015）
        if (user && user.studentNumber === reviewed.studentNumber)
          signIn({ ...user, sellerStatus: reviewed.status })
        const index = cards.findIndex((c) => c.id === selectedId)
        const rest = cards.filter((c) => c.id !== selectedId)
        setCards(rest)
        select(rest[Math.min(index, rest.length - 1)]?.id ?? null)
        setNotice({
          message: `${reviewed.studentNumber} を${doneMessages[decision]}`,
          tone: 'done',
        })
      } catch (err) {
        if (
          isApiError(err) &&
          (err.code === 'invalid_status_transition' || err.code === 'not_found')
        ) {
          reload()
          setNotice({ message: err.message, tone: 'alert' })
        } else {
          setDecisionError(messageOf(err))
        }
      } finally {
        setDeciding(null)
      }
    },
    [cards, selectedId, user, signIn, select, reload],
  )

  const selected = cards?.find((c) => c.id === selectedId)

  return {
    cards,
    loadError,
    reload,
    selected,
    select,
    image,
    views,
    viewsError,
    decide,
    deciding,
    decisionError,
    notice,
  }
}
