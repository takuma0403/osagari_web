import { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router'
import { AppHeader } from '../../../components/AppHeader/AppHeader'
import { Badge } from '../../../components/Badge/Badge'
import { Button, ButtonLink } from '../../../components/Button/Button'
import {
  DefinitionList,
  type DefinitionItem,
} from '../../../components/DefinitionList/DefinitionList'
import { Icon } from '../../../components/Icon/Icon'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import { Toast } from '../../../components/Toast/Toast'
import { formatPrice } from '../../../lib/format'
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
import { useItem } from '../hooks/useItem'
import { itemPath, itemShareText } from '../itemDisplay'
import styles from './DetailPage.module.css'

/** コピーできたことを知らせるトーストを出しておく時間 */
const TOAST_MS = 2000
const copyFailedMessage = 'コピーできませんでした。お使いのブラウザでは使えない可能性があります'

/** 取引中・取引完了は売り切れ（購入希望は出せない。ADR 0018） */
const isSoldOut = (item: PublicItem) => item.status === 'in_deal' || item.status === 'completed'

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

/** 下部の主ボタン。本人の出品と売り切れは押せない（編集の導線は商品IDが要るため別 issue） */
function PrimaryAction({ item }: { item: PublicItem }) {
  if (item.isOwn) {
    return (
      <Button fullWidth disabled>
        自分の出品です
      </Button>
    )
  }
  if (isSoldOut(item)) {
    return (
      <Button fullWidth disabled>
        売り切れ
      </Button>
    )
  }
  return (
    <ButtonLink to={`${itemPath(item.shareId)}/request`} fullWidth>
      購入希望を出す
    </ButtonLink>
  )
}

/**
 * 商品詳細（BROWSE、Detail・DetailShare）。共有リンク /items/:shareId で開く。
 * 「リンクをコピー」で共有シートを開き、商品名・価格・状態を添えた文をコピーする（FE）。
 * 出品者は学部・学科と入学年度だけを出し、学籍番号は取引成立まで出さない（DEAL-1）。
 * 「購入希望 n件」は件数が API にまだないため出さない（ADR 0018）。
 */
export function DetailPage() {
  const { shareId = '' } = useParams()
  const navigate = useNavigate()
  const location = useLocation()
  const { state, reload } = useItem(shareId)
  const [shareOpen, setShareOpen] = useState(false)
  const [copyError, setCopyError] = useState<string | null>(null)
  const [copiedAt, setCopiedAt] = useState<number | null>(null)

  // コピーできたことのトーストは少しして消す
  useEffect(() => {
    if (copiedAt === null) return
    const timer = setTimeout(() => setCopiedAt(null), TOAST_MS)
    return () => clearTimeout(timer)
  }, [copiedAt])

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
      () => setCopiedAt(Date.now()),
      () => setCopyError(copyFailedMessage),
    )
  }

  const closeShare = () => {
    setShareOpen(false)
    setCopyError(null)
  }

  return (
    <div className={styles.page}>
      <ItemGallery imageUrls={item.images.map((i) => i.url)} tint={categoryTint[item.category]}>
        <button type="button" className={styles.back} aria-label="戻る" onClick={back}>
          <Icon name="back" />
        </button>
      </ItemGallery>
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
        <DefinitionList items={detailRows(item)} boxed />
        {item.description && <p className={styles.description}>{item.description}</p>}
        <div className={styles.seller}>
          <span className={styles.avatar} aria-hidden="true">
            {item.seller.departmentName?.charAt(0) ?? '?'}
          </span>
          <div className={styles.sellerText}>
            <span className={styles.sellerName}>
              {item.seller.departmentName ?? '学部不明'} · {item.seller.admissionYear}年度入学
            </span>
            <span className={styles.note}>学籍番号は取引成立後に表示されます</span>
          </div>
        </div>
      </div>
      <div className={styles.actions}>
        <Button variant="ghost" icon="copy" onClick={() => setShareOpen(true)}>
          リンクをコピー
        </Button>
        <PrimaryAction item={item} />
      </div>
      <ShareSheet open={shareOpen} onCopy={copy} onClose={closeShare} error={copyError} />
      <Toast message={copiedAt !== null ? 'リンクをコピーしました' : undefined} />
    </div>
  )
}
