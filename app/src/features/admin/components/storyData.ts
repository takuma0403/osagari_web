import type { AdminStudentCard, AdminUser } from '../api'

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
