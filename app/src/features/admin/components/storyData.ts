import type { AdminItem, AdminStudentCard, AdminTransaction, AdminUser } from '../api'

/** story で使う審査待ちの学生証（AdminCard のサンプルと同じ値） */
export const sampleCards: AdminStudentCard[] = [
  ['c1', '26141023', '経済学部 経済学科', '2026-10-06T01:12:00Z'],
  ['c2', '25238117', '理工学部 理工学科', '2026-10-06T04:40:00Z'],
  ['c3', '26212045', '医学部 看護学科', '2026-10-06T23:55:00Z'],
  ['c4', '72612004', null, '2026-10-08T02:47:00Z'],
].map(([id, studentNumber, departmentName, submittedAt]) => ({
  id: id as string,
  status: 'pending',
  studentNumber: studentNumber as string,
  departmentName,
  admissionYear: 2026,
  expiresOn: '2030-03-31',
  submittedAt: submittedAt as string,
  reviewedAt: null,
}))

/** story で使うユーザー（AdminUsers のサンプルと同じ値） */
export const sampleUsers: AdminUser[] = (
  [
    ['u1', '26141023', '経済学部 経済学科', 'active', 'pending', '2030-03-31', 0],
    ['u2', '25238117', '理工学部 理工学科', 'active', 'approved', '2029-03-31', 4],
    ['u3', '24101032', '教育学部 学校教育課程', 'suspended', 'approved', '2028-03-31', 7],
    ['u4', '26212045', '医学部 看護学科', 'active', 'none', '2030-03-31', 0],
    ['u5', '22238090', '理工学部 理工学科', 'expired', 'approved', '2026-03-31', 1],
    ['u6', '25141077', '経済学部 経済学科', 'active', 'rejected', '2029-03-31', 0],
  ] as const
).map(([id, studentNumber, departmentName, status, sellerStatus, expiresOn, cancelCount]) => ({
  id,
  studentNumber,
  departmentName,
  admissionYear: 2000 + Number(studentNumber.slice(0, 2)),
  expiresOn,
  status,
  sellerStatus,
  cancelCount,
  activeTransactionCount: id === 'u2' ? 1 : 0,
  withdrawalRequestedAt: id === 'u2' ? '2026-10-05T03:00:00Z' : null,
  createdAt: '2026-04-01T00:00:00Z',
}))

/** story で使う出品（AdminDeals のサンプルと同じ値） */
export const sampleItems: AdminItem[] = (
  [
    ['i1', '一人暮らし用 冷蔵庫 2ドア 138L', '25238117', 6000, 'published', '2026-10-07T03:00:00Z'],
    ['i2', '解剖学講義 改訂3版', '24212019', 3500, 'requested', '2026-10-06T03:00:00Z'],
    ['i3', '線形代数入門', '25238117', 1500, 'in_deal', '2026-10-03T03:00:00Z'],
    ['i4', '[停止した出品の例]', '24101032', 2000, 'suspended', '2026-10-02T03:00:00Z'],
    ['i5', '携帯ゲーム機 本体＋ソフト2本', '23141055', 15000, 'completed', '2026-09-25T03:00:00Z'],
  ] as const
).map(([id, title, sellerStudentNumber, price, status, createdAt]) => ({
  id,
  shareId: `s-${id}`,
  title,
  price,
  status,
  sellerStudentNumber,
  thumbnail: null,
  createdAt,
}))

const at = '2026-10-08T05:02:00Z'

/** story で使う取引（AdminDeals のサンプルと同じ値） */
export const sampleTransactions: AdminTransaction[] = (
  [
    ['t1', '線形代数入門', '25238117', '26141023', 1500, 'in_deal', at, null],
    ['t2', '一人暮らし用 冷蔵庫 2ドア 138L', '25238117', '26212045', 6000, 'in_deal', null, null],
    ['t3', '解剖学講義 改訂3版', '24212019', '25141077', 3500, 'requested', null, null],
    ['t4', '携帯ゲーム機 本体＋ソフト2本', '23141055', '24238031', 15000, 'completed', null, null],
    ['t5', '有機化学 演習', '22141008', '26271008', 700, 'cancelled', at, at],
  ] as const
).map(([id, title, seller, buyer, price, status, buyerCancelAgreedAt, sellerCancelAgreedAt]) => ({
  id,
  status,
  item: { shareId: `s-${id}`, title, status: status === 'completed' ? 'completed' : 'in_deal' },
  sellerStudentNumber: seller,
  buyerStudentNumber: buyer,
  price,
  paymentMethod: 'card',
  feeRateBasisPoints: status === 'requested' ? null : 500,
  buyerCancelAgreedAt,
  sellerCancelAgreedAt,
  completedAt: status === 'completed' ? '2026-09-28T03:40:00Z' : null,
  createdAt: '2026-10-03T00:00:00Z',
  updatedAt: at,
}))
