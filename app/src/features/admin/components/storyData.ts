import type { AdminStudentCard } from '../api'

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
