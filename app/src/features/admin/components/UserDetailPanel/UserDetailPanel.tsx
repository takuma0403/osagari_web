import { useState } from 'react'
import { Badge } from '../../../../components/Badge/Badge'
import { Button } from '../../../../components/Button/Button'
import { ConfirmDialog } from '../../../../components/ConfirmDialog/ConfirmDialog'
import { DefinitionList } from '../../../../components/DefinitionList/DefinitionList'
import { TextField } from '../../../../components/TextField/TextField'
import { departmentLabel } from '../../../../lib/department'
import { sellerStatusLabels } from '../../../mypage/myPageDisplay'
import { isUserManageable, userStatusDisplay } from '../../adminDisplay'
import type { AdminUser } from '../../api'
import { DetailPanel, PanelSection } from '../DetailPanel/DetailPanel'
import styles from './UserDetailPanel.module.css'

export type UserAction = 'expiresOn' | 'suspend' | 'unsuspend' | 'delete'

type Props = {
  user: AdminUser
  /** 実行中の操作。実行中はほかの操作を押せない */
  running: UserAction | null
  /** 操作の失敗 */
  error: string | null
  onChangeExpiresOn: (expiresOn: string) => void
  /** 確認のあと停止・削除する。終えたか（useAdminAction の run）を返し、終えたら確認を閉じる */
  onSuspend: () => Promise<boolean>
  onUnsuspend: () => void
  onDelete: () => Promise<boolean>
}

/**
 * ユーザーの詳細と操作（AdminUsers。ADMIN, AUTH-3、ADR 0034）。
 * 有効期限の変更と退学による削除は有効・停止のときだけ。停止中は「アカウントを停止」の位置に「停止を解除」を出す（ADR 0035）。
 * 停止・削除は取り消しの効かない後処理があるので、確認してから行う。
 */
export function UserDetailPanel({
  user,
  running,
  error,
  onChangeExpiresOn,
  onSuspend,
  onUnsuspend,
  onDelete,
}: Props) {
  const [expiresOn, setExpiresOn] = useState(user.expiresOn)
  const [confirming, setConfirming] = useState<'suspend' | 'delete' | null>(null)
  const status = userStatusDisplay[user.status]
  const manageable = isUserManageable(user)
  const busy = running !== null

  const confirm = async () => {
    const ok = await (confirming === 'suspend' ? onSuspend() : onDelete())
    if (ok) setConfirming(null)
  }

  return (
    <DetailPanel label="ユーザーの詳細">
      <div className={styles.head}>
        <span className={styles.number}>{user.studentNumber}</span>
        <Badge tone={status.tone}>{status.label}</Badge>
        {user.withdrawalRequestedAt && user.status !== 'deleted' && (
          <Badge tone="warning">退学申請あり</Badge>
        )}
      </div>
      <DefinitionList
        items={[
          { term: '学部', description: departmentLabel(user.departmentName) },
          { term: '出品者の承認', description: sellerStatusLabels[user.sellerStatus] },
          { term: '進行中の取引', description: `${user.activeTransactionCount}件` },
          {
            term: 'キャンセル回数',
            description: <span className={styles.num}>{user.cancelCount}</span>,
          },
        ]}
      />
      {manageable ? (
        <>
          <PanelSection row note="延長の仕組みができるまでの暫定の操作です。">
            <TextField
              label="有効期限の変更"
              type="date"
              size="sm"
              numeric
              value={expiresOn}
              onChange={(e) => setExpiresOn(e.target.value)}
              inputClassName={styles.date}
            />
            <Button
              size="sm"
              onClick={() => onChangeExpiresOn(expiresOn)}
              disabled={busy || expiresOn === '' || expiresOn === user.expiresOn}
            >
              {running === 'expiresOn' ? '変更中…' : '変更する'}
            </Button>
          </PanelSection>
          <PanelSection row error={confirming ? null : error}>
            {user.status === 'suspended' ? (
              <Button variant="ghost" size="sm" onClick={onUnsuspend} disabled={busy}>
                {running === 'unsuspend' ? '解除中…' : '停止を解除'}
              </Button>
            ) : (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setConfirming('suspend')}
                disabled={busy}
              >
                アカウントを停止
              </Button>
            )}
            <Button
              variant="danger"
              size="sm"
              onClick={() => setConfirming('delete')}
              disabled={busy}
            >
              退学によりアカウントを削除
            </Button>
          </PanelSection>
        </>
      ) : (
        <PanelSection note="失効・削除したアカウントは操作できません。" />
      )}
      <ConfirmDialog
        open={confirming === 'suspend'}
        title="アカウントを停止しますか？"
        confirmLabel={running === 'suspend' ? '停止中…' : '停止する'}
        onConfirm={confirm}
        cancelLabel="やめる"
        onCancel={() => setConfirming(null)}
        confirmDisabled={busy}
      >
        <span className={styles.num}>{user.studentNumber}</span>{' '}
        はログインできなくなり、出品は一覧・商品ページに表示されなくなります。取引はそのまま残ります。停止はあとで解除できます。
        {error && (
          <p role="alert" className={styles.error}>
            {error}
          </p>
        )}
      </ConfirmDialog>
      <ConfirmDialog
        open={confirming === 'delete'}
        title="退学によりアカウントを削除しますか？"
        confirmLabel={running === 'delete' ? '削除中…' : '削除する'}
        onConfirm={confirm}
        cancelLabel="やめる"
        onCancel={() => setConfirming(null)}
        confirmDisabled={busy}
      >
        <span className={styles.num}>{user.studentNumber}</span>{' '}
        の出品をすべて取り下げ、購入希望は期限切れにし、取引中の取引はキャンセルして代金を購入者に返金（モック）します。削除は元に戻せません。
        {error && (
          <p role="alert" className={styles.error}>
            {error}
          </p>
        )}
      </ConfirmDialog>
    </DetailPanel>
  )
}
