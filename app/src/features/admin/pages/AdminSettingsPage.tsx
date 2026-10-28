import { useEffect, useId, useState, type FormEvent } from 'react'
import { Button } from '../../../components/Button/Button'
import { DefinitionList } from '../../../components/DefinitionList/DefinitionList'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import { TextField } from '../../../components/TextField/TextField'
import { Toast } from '../../../components/Toast/Toast'
import { formatFeeRate } from '../../../lib/fee'
import { formatShortDateTime } from '../../../lib/format'
import { parseFeeRatePercent } from '../adminDisplay'
import { getAdminFeeRate, updateFeeRate, type AdminFeeRate } from '../api'
import { AdminLoadState } from '../components/AdminLoadState/AdminLoadState'
import { messageOf } from '../errorMessage'
import common from './adminPage.module.css'
import styles from './AdminSettingsPage.module.css'

const invalidMessage = '0〜100 の範囲で、小数点以下2桁までで入力してください。'

/**
 * 管理：設定（ADMIN, PAY-2）。デザインは AdminSettings.dc.html。
 * 現在の手数料率と最後に変えた日時・管理者を出し、新しい料率（%）をベーシスポイントにして保存する（ADR 0034）。
 * 保存後に成立する取引から新しい料率が使われ、成立済みの取引の料率は変わらない（ADR 0003）。
 */
export function AdminSettingsPage() {
  const [feeRate, setFeeRate] = useState<AdminFeeRate>()
  const [loadError, setLoadError] = useState<string | null>(null)
  const [reloadKey, setReloadKey] = useState(0)
  const [input, setInput] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [notice, setNotice] = useState<string>()
  const titleId = useId()
  const helpId = useId()
  const errorId = useId()

  useEffect(() => {
    const controller = new AbortController()
    getAdminFeeRate(controller.signal)
      .then(setFeeRate)
      .catch((err: unknown) => {
        if (!controller.signal.aborted) setLoadError(messageOf(err))
      })
    return () => controller.abort()
  }, [reloadKey])

  // 保存の結果のトーストは少しだけ出して消す
  useEffect(() => {
    if (!notice) return
    const timer = setTimeout(() => setNotice(undefined), 3000)
    return () => clearTimeout(timer)
  }, [notice])

  const reload = () => {
    setLoadError(null)
    setReloadKey((key) => key + 1)
  }

  const save = async (e: FormEvent) => {
    e.preventDefault()
    const basisPoints = parseFeeRatePercent(input)
    if (basisPoints === null) {
      setError(invalidMessage)
      return
    }
    setSaving(true)
    setError(null)
    try {
      const saved = await updateFeeRate(basisPoints)
      setFeeRate(saved)
      setInput('')
      setNotice(`手数料率を${formatFeeRate(saved.feeRateBasisPoints)}に変更しました`)
    } catch (err) {
      setError(messageOf(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <>
      <h1 className={common.title}>設定</h1>
      {feeRate ? (
        <section aria-labelledby={titleId} className={styles.card}>
          <div className={styles.intro}>
            <h2 id={titleId} className={styles.cardTitle}>
              手数料率
            </h2>
            <p className={styles.lead}>
              出品者の売上 = 商品価格 − 手数料。取引ごとに、適用した料率が記録されます。
            </p>
          </div>
          <DefinitionList
            items={[
              {
                term: '現在の料率',
                description: (
                  <span className={styles.rate}>{formatFeeRate(feeRate.feeRateBasisPoints)}</span>
                ),
              },
              {
                term: '最終更新',
                // 管理者がまだ変えていなければ、変更者はいない（初期値）
                description: `${formatShortDateTime(feeRate.updatedAt)} · ${feeRate.updatedByStudentNumber ?? '初期値'}`,
              },
            ]}
          />
          <form className={styles.form} onSubmit={save} noValidate>
            <div className={styles.row}>
              <TextField
                label="新しい料率"
                inputMode="decimal"
                numeric
                suffix="%"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                invalid={error !== null}
                aria-describedby={error ? `${errorId} ${helpId}` : helpId}
                inputClassName={styles.input}
              />
              <Button type="submit" size="md" disabled={saving}>
                {saving ? '保存中…' : '保存する'}
              </Button>
            </div>
            {error && (
              <div id={errorId} role="alert" className={styles.error}>
                {error}
              </div>
            )}
            <span id={helpId} className={styles.help}>
              小数点以下2桁まで（0.01% 単位）。保存後に成立する取引から新しい料率が使われます。
            </span>
          </form>
          <InfoBox tone="warning" icon="alert">
            フェーズ1の決済はモックです。実際のお金は動きません。
          </InfoBox>
        </section>
      ) : (
        <AdminLoadState error={loadError} onReload={reload} />
      )}
      <Toast message={notice} />
    </>
  )
}
