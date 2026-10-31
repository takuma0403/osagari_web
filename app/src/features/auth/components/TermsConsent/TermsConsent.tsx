import { useId, useState } from 'react'
import { BottomSheet } from '../../../../components/BottomSheet/BottomSheet'
import { Button } from '../../../../components/Button/Button'
import { LegalDocument } from '../../../legal/components/LegalDocument/LegalDocument'
import { legalTitles, type LegalDoc } from '../../../legal/legalDocs'
import styles from './TermsConsent.module.css'

type Props = {
  checked: boolean
  onChange: (checked: boolean) => void
}

/**
 * 新規登録での利用規約・プライバシーポリシーへの同意（AUTH-1）。
 * 両方の文書をシートで開くまではチェックできない。開いたかどうか・同意したかどうかは画面の中だけで持ち、記録しない
 */
export function TermsConsent({ checked, onChange }: Props) {
  const hintId = useId()
  const [opened, setOpened] = useState<ReadonlySet<LegalDoc>>(new Set())
  const [showing, setShowing] = useState<LegalDoc | null>(null)
  const ready = opened.has('terms') && opened.has('privacy')

  const open = (doc: LegalDoc) => {
    setShowing(doc)
    setOpened((prev) => new Set(prev).add(doc))
  }
  const close = () => setShowing(null)

  return (
    <div className={styles.consent}>
      <p className={styles.lead}>
        <button type="button" className={styles.docLink} onClick={() => open('terms')}>
          利用規約
        </button>
        と
        <button type="button" className={styles.docLink} onClick={() => open('privacy')}>
          プライバシーポリシー
        </button>
        を開いて内容を確認してください。
      </p>
      <label className={[styles.check, !ready && styles.disabled].filter(Boolean).join(' ')}>
        <input
          type="checkbox"
          checked={checked}
          disabled={!ready}
          aria-describedby={ready ? undefined : hintId}
          onChange={(e) => onChange(e.target.checked)}
        />
        利用規約とプライバシーポリシーを確認し、同意します
      </label>
      {!ready && (
        <p id={hintId} className={styles.hint}>
          両方の文書を開くとチェックできます
        </p>
      )}
      <BottomSheet
        open={showing !== null}
        onClose={close}
        title={showing ? legalTitles[showing] : ''}
        action={
          <Button variant="text" size="sm" onClick={close}>
            閉じる
          </Button>
        }
      >
        {/* 本文中の規約のリンクは、画面を移らずにシートの文書を切り替える（入力中の内容を残すため） */}
        {showing && <LegalDocument doc={showing} onOpenDoc={open} />}
      </BottomSheet>
    </div>
  )
}
