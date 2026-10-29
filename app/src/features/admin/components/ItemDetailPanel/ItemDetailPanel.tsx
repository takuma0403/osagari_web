import { useState } from 'react'
import { Button } from '../../../../components/Button/Button'
import { ConfirmDialog } from '../../../../components/ConfirmDialog/ConfirmDialog'
import { DefinitionList } from '../../../../components/DefinitionList/DefinitionList'
import { ItemPhoto } from '../../../../components/ItemPhoto/ItemPhoto'
import { formatPrice } from '../../../../lib/format'
import { isItemSuspendable, itemStatusDisplay } from '../../adminDisplay'
import type { AdminItem } from '../../api'
import { DetailPanel, PanelSection } from '../DetailPanel/DetailPanel'
import styles from './ItemDetailPanel.module.css'

type Props = {
  item: AdminItem
  running: boolean
  error: string | null
  /** 確認のあと停止する。終えたか（useAdminAction の run）を返し、終えたら確認を閉じる */
  onSuspend: () => Promise<boolean>
}

/**
 * 出品の詳細と停止（AdminDeals の出品。ADMIN, ITEM-1、ADR 0034）。
 * 停止できるのは下書き・公開中・購入希望あり・取引中だけで、ほかのステータスではボタンを押せない。停止の解除はない。
 */
export function ItemDetailPanel({ item, running, error, onSuspend }: Props) {
  const [confirming, setConfirming] = useState(false)

  const confirm = async () => {
    if (await onSuspend()) setConfirming(false)
  }

  return (
    <DetailPanel label="出品の詳細">
      {/* 管理の出品はカテゴリを持たないので、写真がなければデザインの写真の枠と同じ淡い青の地にする */}
      <ItemPhoto imageUrl={item.thumbnail?.url} tint="appliance" className={styles.photo}>
        {!item.thumbnail && <span className={styles.noPhoto}>写真なし</span>}
      </ItemPhoto>
      <h2 className={styles.title}>{item.title}</h2>
      <DefinitionList
        items={[
          {
            term: '出品者',
            description: <span className={styles.strongNum}>{item.sellerStudentNumber}</span>,
          },
          {
            term: '価格',
            description:
              item.price === null ? (
                '未入力'
              ) : (
                <span className={styles.strongNum}>{formatPrice(item.price)}</span>
              ),
          },
          { term: 'ステータス', description: itemStatusDisplay[item.status].label },
        ]}
      />
      <PanelSection
        note="停止すると一覧・検索に表示されなくなります。"
        error={confirming ? null : error}
      >
        <Button
          variant="danger"
          size="sm"
          onClick={() => setConfirming(true)}
          disabled={running || !isItemSuspendable(item)}
        >
          出品を停止
        </Button>
      </PanelSection>
      <ConfirmDialog
        open={confirming}
        title="出品を停止しますか？"
        confirmLabel={running ? '停止中…' : '停止する'}
        onConfirm={confirm}
        cancelLabel="やめる"
        onCancel={() => setConfirming(false)}
        confirmDisabled={running}
      >
        {`「${item.title}」は一覧・検索に表示されなくなります。`}
        購入希望があればすべて却下し、取引中なら取引はそのまま続きます。停止は解除できません。
        {error && (
          <p role="alert" className={styles.error}>
            {error}
          </p>
        )}
      </ConfirmDialog>
    </DetailPanel>
  )
}
