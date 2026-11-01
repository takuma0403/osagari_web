import {
  filterChips,
  formatPriceRange,
  parseSearchConditions,
  toSearchParams,
} from './searchConditions'

describe('parseSearchConditions', () => {
  it('URL のクエリを検索条件にする', () => {
    const params = new URLSearchParams(
      'keyword=線形代数&category=textbook&courseName=線形代数学I&priceMin=500&priceMax=2000&campus=honjo&condition=good',
    )
    expect(parseSearchConditions(params)).toEqual({
      keyword: '線形代数',
      category: 'textbook',
      courseName: '線形代数学I',
      priceMin: 500,
      priceMax: 2000,
      campus: 'honjo',
      condition: 'good',
    })
  })

  it('知らない値・数字でない価格・空白だけの文字は条件にしない', () => {
    const params = new URLSearchParams(
      'keyword=%20%20&category=game&priceMin=abc&priceMax=-1&campus=tokyo&condition=bad',
    )
    expect(parseSearchConditions(params)).toEqual({})
  })

  it('前後の空白を除き、100文字までにする', () => {
    const params = new URLSearchParams({ keyword: ` ${'あ'.repeat(120)} ` })
    expect(parseSearchConditions(params).keyword).toBe('あ'.repeat(100))
  })
})

describe('toSearchParams', () => {
  it('決まった順に並べ、指定のない条件は省く', () => {
    expect(
      toSearchParams({ campus: 'nabeshima', keyword: '解剖学', priceMax: 3000 }).toString(),
    ).toBe('keyword=%E8%A7%A3%E5%89%96%E5%AD%A6&priceMax=3000&campus=nabeshima')
    expect(toSearchParams({ keyword: '' }).toString()).toBe('')
  })

  it('parseSearchConditions と往復できる', () => {
    const conditions = { keyword: '線形代数', category: 'textbook', priceMin: 0 } as const
    expect(parseSearchConditions(toSearchParams(conditions))).toEqual(conditions)
  })
})

describe('formatPriceRange', () => {
  it('下限・上限の片方だけでも表示できる', () => {
    expect(formatPriceRange(1000, 2000)).toBe('¥1,000〜¥2,000')
    expect(formatPriceRange(1000)).toBe('¥1,000〜')
    expect(formatPriceRange(undefined, 2000)).toBe('〜¥2,000')
  })
})

describe('filterChips', () => {
  it('キーワード以外の条件をチップにし、外したときの条件を持つ', () => {
    const chips = filterChips({
      keyword: '線形代数',
      category: 'textbook',
      priceMax: 2000,
      campus: 'honjo',
    })
    expect(chips.map((c) => c.label)).toEqual(['教科書・参考書', '〜¥2,000', '本庄'])
    expect(chips[1].without).toEqual({ keyword: '線形代数', category: 'textbook', campus: 'honjo' })
  })

  it('授業名と状態もチップにする', () => {
    const chips = filterChips({ courseName: '線形代数学I', condition: 'new_like' })
    expect(chips.map((c) => c.label)).toEqual(['授業名：線形代数学I', '新品同様'])
  })
})
