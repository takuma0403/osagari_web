/** 大学のメールアドレスのドメイン（AUTHN-2）。利用者は @ より前の学籍番号8桁だけを入力する */
export const UNIVERSITY_EMAIL_DOMAIN = '@edu.cc.saga-u.ac.jp'

export const STUDENT_ID_LENGTH = 8

/**
 * 入力された文字を学籍番号の数字だけにする。全角数字は半角にし、アドレスごと貼り付けられたら @ より前を使う。
 * 最終的な検証はサーバーが行う（AUTHN-2）。
 */
export function normalizeStudentId(input: string): string {
  return input.normalize('NFKC').split('@')[0].replace(/\D/g, '').slice(0, STUDENT_ID_LENGTH)
}
