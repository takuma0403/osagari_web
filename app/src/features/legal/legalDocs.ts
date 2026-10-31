export type LegalDoc = 'terms' | 'privacy'

/** 利用規約・プライバシーポリシーの画面・シートの見出し */
export const legalTitles: Record<LegalDoc, string> = {
  terms: '利用規約',
  privacy: 'プライバシーポリシー',
}
