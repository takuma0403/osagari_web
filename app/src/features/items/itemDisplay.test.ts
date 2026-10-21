import type { ItemSummary } from './api'
import { cardProps, rowProps } from './itemDisplay'

const base: ItemSummary = {
  shareId: 'abc123',
  status: 'published',
  title: '線形代数入門',
  category: 'textbook',
  condition: 'good',
  price: 1500,
  campus: 'honjo',
  thumbnail: {
    objectKey: 'u1/k1',
    url: 'http://localhost:4443/u1/k1?sig=x',
    expiresAt: '2026-10-22T05:00:00Z',
  },
  courseName: '線形代数学I',
  hasWriting: false,
  publishedAt: '2026-10-22T04:00:00Z',
}

describe('cardProps', () => {
  it('商品ページへのリンク・表示名・写真の地の色・写真を渡す', () => {
    expect(cardProps({ ...base, category: 'furniture_appliance', campus: 'nabeshima' })).toEqual({
      to: '/items/abc123',
      title: '線形代数入門',
      price: 1500,
      categoryLabel: '家具・家電',
      tint: 'furniture',
      campus: '鍋島',
      condition: '目立った傷なし',
      imageUrl: 'http://localhost:4443/u1/k1?sig=x',
    })
  })

  it('写真がなければ imageUrl を渡さない', () => {
    expect(cardProps({ ...base, thumbnail: null }).imageUrl).toBeUndefined()
  })
})

describe('rowProps', () => {
  it('教科書は授業名と書き込みの有無を出す', () => {
    const row = rowProps({ ...base, hasWriting: true })
    expect(row.subtitle).toBe('線形代数学I')
    expect(row.note).toBe('書き込みあり')
  })

  it('教科書でも授業名・書き込みが分からなければカテゴリ名と状態を出す', () => {
    const row = rowProps({ ...base, courseName: null, hasWriting: null })
    expect(row.subtitle).toBe('教科書・参考書')
    expect(row.note).toBe('目立った傷なし')
  })

  it('教科書以外はカテゴリ名と状態を出す', () => {
    const row = rowProps({
      ...base,
      category: 'game_other',
      condition: 'fair',
      courseName: null,
      hasWriting: null,
    })
    expect(row).toMatchObject({ subtitle: 'ゲーム・その他', note: 'やや傷あり', tint: 'game' })
  })
})
