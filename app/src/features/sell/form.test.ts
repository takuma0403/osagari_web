import type { OwnItem } from './api'
import { emptySellForm, formFromItem, toItemInput, validateSellForm, type SellForm } from './form'

const filled: SellForm = {
  ...emptySellForm,
  title: '線形代数入門',
  category: 'textbook',
  condition: 'good',
  price: '1500',
  description: '書き込みはありません。',
  campus: 'honjo',
}

describe('validateSellForm', () => {
  it('下書きはタイトルだけ必須', () => {
    expect(validateSellForm({ ...emptySellForm, title: 'a' }, 0, false)).toEqual({})
    expect(validateSellForm(emptySellForm, 0, false)).toEqual({
      title: 'タイトルを入力してください',
    })
  })

  it('公開はすべての共通項目と写真1枚以上が必須', () => {
    expect(validateSellForm(filled, 1, true)).toEqual({})
    expect(validateSellForm({ ...emptySellForm, title: 'a' }, 0, true)).toEqual({
      photos: '写真を1枚以上追加してください',
      category: 'カテゴリを選んでください',
      condition: '状態を選んでください',
      price: '価格を入力してください',
      description: '説明を入力してください',
      campus: '受け渡し希望キャンパスを選んでください',
    })
  })

  it.each([
    [{ title: 'あ'.repeat(101) }, 'title', 'タイトルは100文字以内で入力してください'],
    [{ price: '99' }, 'price', '価格は100円から100,000円の範囲で入力してください'],
    [{ price: '100001' }, 'price', '価格は100円から100,000円の範囲で入力してください'],
    [{ description: 'あ'.repeat(2001) }, 'description', '説明は2,000文字以内で入力してください'],
    [{ courseName: 'あ'.repeat(101) }, 'courseName', '授業名は100文字以内で入力してください'],
    [{ isbn: '9784000000002' }, 'isbn', 'ISBN を確認してください（10桁か13桁）'],
  ] as const)('入力した項目の範囲を確かめる %o', (patch, field, message) => {
    expect(validateSellForm({ ...filled, ...patch }, 1, false)).toEqual({ [field]: message })
  })

  it('カテゴリと合わない追加項目は確かめない（送らないため）', () => {
    expect(validateSellForm({ ...filled, category: 'game_other', isbn: 'x' }, 1, true)).toEqual({})
  })
})

describe('toItemInput', () => {
  it('空の任意項目を省き、ISBN のハイフンを除く', () => {
    expect(
      toItemInput({ ...filled, isbn: '978-4-00-000000-0', courseName: '  ' }, ['u1/a']),
    ).toEqual({
      title: '線形代数入門',
      category: 'textbook',
      condition: 'good',
      price: 1500,
      description: '書き込みはありません。',
      campus: 'honjo',
      imageObjectKeys: ['u1/a'],
      textbook: { isbn: '9784000000000', hasWriting: false },
      furnitureAppliance: undefined,
    })
  })

  it('家具・家電では家具・家電の追加項目だけを送る', () => {
    const input = toItemInput(
      { ...filled, category: 'furniture_appliance', isbn: '978', size: '幅60cm' },
      [],
    )
    expect(input.textbook).toBeUndefined()
    expect(input.furnitureAppliance).toEqual({ size: '幅60cm', operationChecked: false })
  })
})

describe('formFromItem', () => {
  it('null の項目は空にする', () => {
    const item = {
      title: '下書き',
      category: null,
      condition: null,
      price: null,
      description: null,
      campus: null,
      textbook: null,
      furnitureAppliance: null,
    } as unknown as OwnItem
    expect(formFromItem(item)).toEqual({ ...emptySellForm, title: '下書き' })
  })
})
