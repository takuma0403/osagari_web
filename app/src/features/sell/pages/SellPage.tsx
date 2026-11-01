import { useEffect, useState, type FormEvent, type ReactNode } from 'react'
import { useLocation, useNavigate, useParams } from 'react-router'
import { AppHeader } from '../../../components/AppHeader/AppHeader'
import { Button, ButtonLink } from '../../../components/Button/Button'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import { SelectField } from '../../../components/SelectField/SelectField'
import { TextAreaField } from '../../../components/TextAreaField/TextAreaField'
import { TextField } from '../../../components/TextField/TextField'
import { Toast } from '../../../components/Toast/Toast'
import { TwoColumn } from '../../../components/TwoColumn/TwoColumn'
import { estimateProceeds, formatFeeRate } from '../../../lib/fee'
import { formatPrice } from '../../../lib/format'
import { useSession } from '../../auth/session'
import type { OwnItem } from '../api'
import { CampusPicker } from '../components/CampusPicker/CampusPicker'
import { CategoryDetails } from '../components/CategoryDetails/CategoryDetails'
import { PhotoPicker } from '../components/PhotoPicker/PhotoPicker'
import { categoryOptions, conditionOptions } from '../../../lib/itemOptions'
import {
  ITEM_DESCRIPTION_MAX,
  ITEM_TITLE_MAX,
  parsePrice,
  ITEM_PRICE_MAX,
  ITEM_PRICE_MIN,
} from '../form'
import { useItemPhotos } from '../hooks/useItemPhotos'
import { useSellForm, type SellAction } from '../hooks/useSellForm'
import styles from './SellPage.module.css'

/** 画面を移ったあとに出すトースト（location.state） */
type SellLocationState = { toast?: string } | null

const TOAST_MS = 3000

/**
 * 出品・編集（ITEM, STORAGE, PAY-2）。デザインは Sell.dc.html。`/sell` は新規、`/sell/:itemId` は自分の商品の編集。
 * 商品ごとに状態を持ち直すよう、商品IDを key にして中身を作り直す。
 */
export function SellPage() {
  const { itemId } = useParams()
  const { user } = useSession()

  if (user?.sellerStatus !== 'approved')
    return <NotApproved pending={user?.sellerStatus === 'pending'} />
  return <SellEditor key={itemId ?? 'new'} itemId={itemId} />
}

/** 学生証が承認されていないときの案内（CARD）。出品の API も 403 seller_not_approved を返す */
function NotApproved({ pending }: { pending: boolean }) {
  return (
    <div className={styles.page}>
      <AppHeader title="出品する" backTo="/" closable />
      <div className={styles.notice}>
        <InfoBox tone="warning" icon="lock">
          {pending
            ? '学生証の審査が終わると出品できるようになります。'
            : '出品するには学生証の登録が必要です。管理者が学籍番号を確認したら出品できるようになります。'}
        </InfoBox>
        <ButtonLink to="/mypage/student-card" variant={pending ? 'ghost' : 'primary'} fullWidth>
          {pending ? '審査状況を見る' : '学生証を登録する'}
        </ButtonLink>
      </div>
    </div>
  )
}

/** 編集する商品を読み込み、読み込めたら入力欄を出す */
function SellEditor({ itemId }: { itemId: string | undefined }) {
  const sell = useSellForm(itemId)
  const editing = Boolean(itemId)
  const title = editing ? '出品を編集' : '出品する'
  const header = <AppHeader title={title} backTo={editing ? '/mypage' : '/'} closable />

  if (sell.loadError) {
    return (
      <div className={styles.page}>
        {header}
        <div className={styles.notice}>
          <InfoBox tone="danger" icon="alert" role="alert">
            <p>{sell.loadError}</p>
            <Button variant="text" size="sm" onClick={sell.reload}>
              もう一度読み込む
            </Button>
          </InfoBox>
        </div>
      </div>
    )
  }
  if (sell.loading) {
    return (
      <div className={styles.page}>
        {header}
        <p className={styles.loading} role="status">
          読み込み中…
        </p>
      </div>
    )
  }

  return <SellFormView sell={sell} editing={editing} title={title} header={header} />
}

type SellFormViewProps = {
  sell: ReturnType<typeof useSellForm>
  editing: boolean
  title: string
  header: ReactNode
}

/** 入力欄と下部のボタン。写真の並びは、この部品を出した時点の商品（編集なら読み込んだ商品）から作る */
function SellFormView({ sell, editing, title, header }: SellFormViewProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const photos = useItemPhotos(sell.item)
  const [toast, setToast] = useState((location.state as SellLocationState)?.toast)
  const [pending, setPending] = useState<SellAction | null>(null)

  useEffect(() => {
    if (!toast) return
    const timer = setTimeout(() => setToast(undefined), TOAST_MS)
    return () => clearTimeout(timer)
  }, [toast])

  const { form, update, errors, item } = sell
  const disabled = sell.submitting || !sell.editable
  const status = item?.status

  async function handleAction(action: SellAction) {
    setPending(action)
    const saved = await sell.submit(action, photos.objectKeys)
    setPending(null)
    if (!saved) return
    if (saved.status === 'published') {
      navigate(`/items/${saved.shareId}`)
    } else if (!editing) {
      navigate(`/sell/${saved.id}`, { replace: true, state: { toast: '下書きを保存しました' } })
    } else {
      setToast('保存しました')
    }
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    void handleAction(status === 'published' ? 'save' : 'publish')
  }

  const price = parsePrice(form.price)
  const feeHint =
    sell.feeRate === undefined ? undefined : (
      <span className={styles.fee}>
        <span>手数料 {formatFeeRate(sell.feeRate)}</span>
        <span>
          受け取り見込み{' '}
          {price !== undefined && price >= ITEM_PRICE_MIN && price <= ITEM_PRICE_MAX
            ? formatPrice(estimateProceeds(price, sell.feeRate).proceeds)
            : '¥—'}
        </span>
      </span>
    )

  return (
    <form className={styles.page} onSubmit={handleSubmit} noValidate>
      {header}
      <fieldset className={styles.body} disabled={disabled}>
        <legend className="visually-hidden">{title}</legend>
        {!sell.editable && status && <NotEditable status={status} />}
        {/* PC 幅では写真（左）と入力欄（右）の2カラム（DS-1） */}
        <TwoColumn
          variant="narrowStart"
          start={
            <PhotoPicker
              photos={photos.photos}
              onAdd={(files) => {
                sell.clearError('photos')
                void photos.add(files)
              }}
              onRemove={photos.remove}
              onMove={photos.move}
              error={photos.error ?? errors.photos}
              disabled={disabled}
            />
          }
          end={
            <>
              <TextField
                label="タイトル"
                size="md"
                value={form.title}
                maxLength={ITEM_TITLE_MAX}
                error={errors.title}
                onChange={(e) => update('title', e.target.value)}
              />
              <div className={styles.pair}>
                <SelectField
                  label="カテゴリ"
                  options={categoryOptions}
                  value={form.category}
                  placeholder="選択してください"
                  error={errors.category}
                  onChange={(v) => update('category', v)}
                />
                <SelectField
                  label="状態"
                  options={conditionOptions}
                  value={form.condition}
                  placeholder="選択してください"
                  error={errors.condition}
                  onChange={(v) => update('condition', v)}
                />
              </div>
              <CategoryDetails form={form} onChange={update} errors={errors} />
              <TextField
                label="価格"
                size="md"
                prefix="¥"
                numeric
                inputMode="numeric"
                autoComplete="off"
                value={form.price}
                error={errors.price}
                hint={feeHint}
                onChange={(e) => update('price', e.target.value.replace(/\D/g, '').slice(0, 6))}
              />
              <CampusPicker
                value={form.campus}
                error={errors.campus}
                onChange={(v) => update('campus', v)}
                disabled={disabled}
              />
              <TextAreaField
                label="説明"
                rows={3}
                value={form.description}
                maxLength={ITEM_DESCRIPTION_MAX}
                error={errors.description}
                onChange={(e) => update('description', e.target.value)}
              />
              {sell.submitError && (
                <InfoBox tone="danger" icon="alert" role="alert">
                  {sell.submitError}
                </InfoBox>
              )}
            </>
          }
        />
      </fieldset>
      {sell.editable && (
        <Actions
          status={status}
          busy={sell.submitting || photos.uploading}
          pending={pending}
          onSave={() => void handleAction('save')}
        />
      )}
      <Toast message={toast} />
    </form>
  )
}

type ActionsProps = {
  status: OwnItem['status'] | undefined
  /** 送信中・写真のアップロード中は押せない */
  busy: boolean
  /** 送信中の操作。押したボタンだけ「保存中…」にする */
  pending: SellAction | null
  onSave: () => void
}

/**
 * 下部のボタン（ITEM, ADR 0016）。新規・下書きは「下書き保存」「公開する」、取り下げは「保存する」「公開する」（再出品）、
 * 公開中は公開のまま「保存する」だけ。公開する・公開中の保存は form の submit で送る。
 */
function Actions({ status, busy, pending, onSave }: ActionsProps) {
  if (status === 'published') {
    return (
      <div className={styles.actions}>
        <Button type="submit" fullWidth disabled={busy}>
          {pending ? '保存中…' : '保存する'}
        </Button>
      </div>
    )
  }
  return (
    <div className={styles.actions}>
      <Button variant="ghost" className={styles.save} disabled={busy} onClick={onSave}>
        {pending === 'save' ? '保存中…' : status === 'withdrawn' ? '保存する' : '下書き保存'}
      </Button>
      <Button type="submit" className={styles.publish} disabled={busy}>
        {pending === 'publish' ? '保存中…' : '公開する'}
      </Button>
    </div>
  )
}

const notEditableMessages: Partial<Record<OwnItem['status'], string>> = {
  requested: '購入希望が入っているため編集できません。',
  in_deal: '取引中のため編集できません。',
  completed: '取引が完了しているため編集できません。',
  suspended: '管理者が停止したため編集できません。',
}

/** 編集できないステータスの案内（ADR 0016） */
function NotEditable({ status }: { status: OwnItem['status'] }) {
  return (
    <InfoBox tone="warning" icon="lock">
      {notEditableMessages[status] ?? 'この商品は編集できません。'}
    </InfoBox>
  )
}
