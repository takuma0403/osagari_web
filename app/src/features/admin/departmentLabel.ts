/** 推定した学部・学科の表示名。学籍番号の学部コードが表にないときは null なので「不明」と出す（AUTH-2） */
export function departmentLabel(departmentName: string | null): string {
  return departmentName ?? '不明（表にないコード）'
}
