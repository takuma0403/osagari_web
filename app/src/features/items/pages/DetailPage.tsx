import { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router'
import { AppHeader } from '../../../components/AppHeader/AppHeader'
import { Avatar } from '../../../components/Avatar/Avatar'
import { Badge } from '../../../components/Badge/Badge'
import { Button, ButtonLink } from '../../../components/Button/Button'
import {
  DefinitionList,
  type DefinitionItem,
} from '../../../components/DefinitionList/DefinitionList'
import { Icon } from '../../../components/Icon/Icon'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import { Toast } from '../../../components/Toast/Toast'
import { TwoColumn } from '../../../components/TwoColumn/TwoColumn'
import { formatPrice, formatShortDateTime } from '../../../lib/format'
import {
  campusOptions,
  categoryOptions,
  categoryTint,
  conditionOptions,
  labelOf,
} from '../../../lib/itemOptions'
import { copyText } from '../../../lib/share'
import type { PublicItem } from '../api'
import { ItemGallery } from '../components/ItemGallery/ItemGallery'
import { ShareSheet } from '../components/ShareSheet/ShareSheet'
import type { RequestSentState } from '../../deals/pages/RequestPage'
import { useItem } from '../hooks/useItem'
import { isSoldOut, itemPath, itemShareText } from '../itemDisplay'
import styles from './DetailPage.module.css'

/** コピーできた・購入希望を送れたことを知らせるトーストを出しておく時間 */
const TOAST_MS = 2000
const copyFailedMessage = 'コピーできませんでした。お使いのブラウザでは使えない可能性があります'

/** 商品の情報の表。値のない項目は出さない */
function detailRows(item: PublicItem): DefinitionItem[] {
  const rows: (DefinitionItem | false | undefined)[] = [
    !!item.textbook?.courseName && { term: '授業名', description: item.textbook.courseName },
    !!item.textbook?.instructor && { term: '担当教員', description: item.textbook.instructor },
    !!item.textbook?.isbn && {
      term: 'ISBN',
      description: <span className={styles.num}>{item.textbook.isbn}</span>,
    },
    item.textbook?.hasWriting != null && {
      term: '書き込み',
      description: item.textbook.hasWriting ? 'あり' : 'なし',
    },
    !!item.furnitureAppliance?.size && {
      term: 'サイズ',
      description: item.furnitureAppliance.size,
    },
    item.furnitureAppliance?.operationChecked != null && {
      term: '動作確認',
      description: item.furnitureAppliance.operationChecked ? '確認済み' : '未確認',
    },
    { term: '状態', description: labelOf(conditionOptions, item.condition) },
    { term: '受け渡し', description: `${labelOf(campusOptions, item.campus)}キャンパス` },
  ]
  return rows.filter((row): row is DefinitionItem => !!row)
}

/**
 * 下部の主ボタン。売り切れは本人でも押せない（取引中・取引完了は編集できない。ADR 0016）。
 * 本人の出品なら編集画面（購入希望ありなら購入希望の一覧）へ進める。商品IDがない応答では行き先が分からないので押せなくする。
 */
function PrimaryAction({ item }: { item: PublicItem }) {
  if (isSoldOut(item)) {
    return (
      <Button className={styles.primary} disabled>
        売り切れ
      </Button>
    )
  }
  if (item.isOwn) {
    if (!item.itemId) {
      return (
        <Button className={styles.primary} disabled>
          自分の出品です
        </Button>
      )
    }
    // 購入希望ありの間は編集できない（ADR 0016）ので、届いている購入希望を見る画面へ進める（DEAL-1）
    if (item.status === 'requested') {
      return (
        <ButtonLink to={`/sell/${item.itemId}/requests`} className={styles.primary}>
          購入希望を見る
        </ButtonLink>
      )
    }
    return (
      <ButtonLink to={`/sell/${item.itemId}`} className={styles.primary}>
        出品を編集する
      </ButtonLink>
    )
  }
  // 同じ商品に承認待ちの購入希望を出している間は、もう出せない（DEAL-1）
  if (item.myRequest) {
    return (
      <Button className={styles.primary} disabled>
        購入希望済み
      </Button>
    )
  }
  return (
    <ButtonLink to={`${itemPath(item.shareId)}/request`} className={styles.primary}>
      購入希望を出す
    </ButtonLink>
  )
}

/**
 * 商品詳細（BROWSE、Detail・DetailShare）。共有リンク /items/:shareId で開く。
 * 「リンクをコピー」で共有シートを開き、商品名・価格・状態を添えた文をコピーする（FE）。
 * 出品者は学部・学科と入学年度だけを出し、学籍番号は取引成立まで出さない（DEAL-1）。
 * 「購入希望 n件」は件数が API にまだないため出さない（ADR 0018）。
 * 自分が承認待ちの購入希望を出している商品は「購入希望済み」にし、承認期限を出す（DEAL-1）。
 * 期限を過ぎても、定期処理で期限切れになるまでは同じ表示のままにする。
 */
export function DetailPage() {
  const { shareId = '' } = useParams()
  const navigate = useNavigate()
  const location = useLocation()
  const { state, reload } = useItem(shareId)
  const [shareOpen, setShareOpen] = useState(false)
  const [copyError, setCopyError] = useState<string | null>(null)
  // 購入希望を送って戻ってきたとき（RequestPage）は、送れたことをトーストで知らせる
  // コピーし直したときも消すまでの時間を数え直すよう、毎回新しいオブジェクトにする
  const [toast, setToast] = useState<{ message: string } | null>(() =>
    (location.state as RequestSentState | null)?.requestSent
      ? { message: '購入希望を送りました' }
      : null,
  )

  // 再読み込みでトーストをもう一度出さないよう、履歴の state を消しておく
  useEffect(() => {
    if ((location.state as RequestSentState | null)?.requestSent) {
      navigate(location.pathname, { replace: true, state: null })
    }
  }, [location.state, location.pathname, navigate])

  // トーストは商品を出してから少しして消す
  const ready = state.status === 'ready'
  useEffect(() => {
    if (toast === null || !ready) return
    const timer = setTimeout(() => setToast(null), TOAST_MS)
    return () => clearTimeout(timer)
  }, [toast, ready])

  // 共有リンクから直接開いたとき（アプリ内の履歴がない）は一覧へ戻る
  const back = () => (location.key === 'default' ? navigate('/') : navigate(-1))

  if (state.status !== 'ready') {
    return (
      <div className={styles.page}>
        <AppHeader onBack={back} />
        <div className={styles.status}>
          {state.status === 'loading' && <p role="status">読み込み中…</p>}
          {state.status === 'notFound' && (
            <>
              <h1 className={styles.statusTitle}>商品が見つかりません</h1>
              <p className={styles.muted}>取り下げられたか、公開が止まっている可能性があります。</p>
              <Link to="/">商品一覧へ戻る</Link>
            </>
          )}
          {state.status === 'error' && (
            <InfoBox tone="danger" icon="alert" role="alert">
              <p>{state.message}</p>
              <Button variant="text" size="sm" onClick={reload}>
                もう一度読み込む
              </Button>
            </InfoBox>
          )}
        </div>
      </div>
    )
  }

  const { item } = state

  const copy = () => {
    setCopyError(null)
    copyText(itemShareText(item, window.location.origin)).then(
      () => setToast({ message: 'リンクをコピーしました' }),
      () => setCopyError(copyFailedMessage),
    )
  }

  const closeShare = () => {
    setShareOpen(false)
    setCopyError(null)
  }

  return (
    <div className={styles.page}>
      {/* PC 幅では写真（左、360px）と商品の情報（右）の2カラム（DS-1） */}
      <div className={styles.main}>
        <TwoColumn
          variant="narrowStart"
          start={
            <ItemGallery
              imageUrls={item.images.map((i) => i.url)}
              tint={categoryTint[item.category]}
              className={styles.gallery}
            >
              <button type="button" className={styles.back} aria-label="戻る" onClick={back}>
                <Icon name="back" />
              </button>
            </ItemGallery>
          }
          end={
            <div className={styles.body}>
              <div className={styles.heading}>
                <div className={styles.badges}>
                  <Badge size="md">{labelOf(categoryOptions, item.category)}</Badge>
                  {isSoldOut(item) && (
                    <Badge tone="neutral" size="md">
                      売り切れ
                    </Badge>
                  )}
                </div>
                <h1 className={styles.title}>{item.title}</h1>
                <p className={styles.price}>{formatPrice(item.price)}</p>
              </div>
              {item.myRequest && !isSoldOut(item) && (
                <InfoBox tone="warning" icon="clock">
                  出品者の承認待ち · {formatShortDateTime(item.myRequest.approvalDeadline)}まで
                </InfoBox>
              )}
              <DefinitionList items={detailRows(item)} boxed />
              {item.description && <p className={styles.description}>{item.description}</p>}
              <div className={styles.seller}>
                <Avatar departmentName={item.seller.departmentName} />
                <div className={styles.sellerText}>
                  <span className={styles.sellerName}>
                    {item.seller.departmentName ?? '学部不明'} · {item.seller.admissionYear}年度入学
                  </span>
                  <span className={styles.note}>学籍番号は取引成立後に表示されます</span>
                </div>
              </div>
            </div>
          }
        />
      </div>
      <div className={styles.actions}>
        <Button variant="ghost" icon="copy" onClick={() => setShareOpen(true)}>
          リンクをコピー
        </Button>
        <PrimaryAction item={item} />
      </div>
      <ShareSheet open={shareOpen} onCopy={copy} onClose={closeShare} error={copyError} />
      <Toast message={toast?.message} />
    </div>
  )
}
