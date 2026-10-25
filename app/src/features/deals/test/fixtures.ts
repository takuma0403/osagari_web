import type { OwnItem } from '../../sell/api'
import type { PurchaseRequest, Transaction } from '../api'

/** 出品者本人の商品（購入希望あり） */
export function ownItem(overrides: Partial<OwnItem> = {}): OwnItem {
  return {
    id: 'i1',
    shareId: 'abc123',
    status: 'requested',
    title: '線形代数入門',
    category: 'textbook',
    condition: 'good',
    price: 1500,
    description: '1年前期で使いました。',
    campus: 'honjo',
    images: [],
    textbook: { courseName: '線形代数学I', isbn: '9784000000000', hasWriting: false },
    furnitureAppliance: null,
    publishedAt: '2026-10-06T00:00:00Z',
    createdAt: '2026-10-06T00:00:00Z',
    updatedAt: '2026-10-06T00:00:00Z',
    ...overrides,
  }
}

/** 出品者に見せる承認待ちの購入希望 */
export function purchaseRequest(overrides: Partial<PurchaseRequest> = {}): PurchaseRequest {
  return {
    transactionId: 'tx1',
    requestedAt: '2026-10-07T03:40:00Z',
    approvalDeadline: '2026-10-09T03:40:00Z',
    buyer: { departmentName: '経済学部', admissionYear: 2026, completedDealCount: 0 },
    ...overrides,
  }
}

/** 取引（既定は購入者から見た承認待ちの購入希望） */
export function transaction(overrides: Partial<Transaction> = {}): Transaction {
  return {
    id: 'tx1',
    role: 'buyer',
    status: 'requested',
    item: { shareId: 'abc123', title: '線形代数入門', status: 'requested' },
    price: 1500,
    paymentMethod: 'card',
    approvalDeadline: '2026-10-09T03:40:00Z',
    feeRateBasisPoints: null,
    fee: null,
    sellerProceeds: null,
    identityDisclosedAt: null,
    sellerAcknowledgedAt: null,
    buyerCancelAgreedAt: null,
    sellerCancelAgreedAt: null,
    completedAt: null,
    handoverMethod: null,
    counterpartStudentNumber: null,
    createdAt: '2026-10-07T03:40:00Z',
    ...overrides,
  }
}

/** 出品者が承認して成立した取引（購入者の学籍番号を含む） */
export function approvedTransaction(overrides: Partial<Transaction> = {}): Transaction {
  return transaction({
    role: 'seller',
    status: 'in_deal',
    item: { shareId: 'abc123', title: '線形代数入門', status: 'in_deal' },
    feeRateBasisPoints: 500,
    fee: 75,
    sellerProceeds: 1425,
    identityDisclosedAt: '2026-10-08T03:40:00Z',
    counterpartStudentNumber: '26141023',
    ...overrides,
  })
}

/** 購入者から見た取引中の取引（出品者の学籍番号を含む） */
export function buyerDealTransaction(overrides: Partial<Transaction> = {}): Transaction {
  return transaction({
    status: 'in_deal',
    item: { shareId: 'abc123', title: '線形代数入門', status: 'in_deal' },
    identityDisclosedAt: '2026-10-08T03:40:00Z',
    counterpartStudentNumber: '25238117',
    ...overrides,
  })
}

type Handler = (init?: RequestInit) => Response | Promise<Response>

/** 「メソッド URL」（例 `POST /api/transactions/tx1/approve`）ごとに応答を返し分ける fetch を差し込む */
export function stubDealsFetch(handlers: Record<string, Handler>) {
  const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
    const key = `${init?.method ?? 'GET'} ${url}`
    const handler = handlers[key]
    if (!handler) throw new Error(`想定していない呼び出し: ${key}`)
    return handler(init)
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

/** 呼び出しのうち、指定した「メソッド URL」の本文（JSON）を取り出す */
export function bodyOf(fetchMock: ReturnType<typeof stubDealsFetch>, key: string): unknown {
  const call = fetchMock.mock.calls.find(([url, init]) => `${init?.method ?? 'GET'} ${url}` === key)
  if (!call) throw new Error(`呼ばれていない: ${key}`)
  return JSON.parse(String(call[1]?.body))
}
