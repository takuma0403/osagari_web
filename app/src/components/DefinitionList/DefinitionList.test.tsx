import { render, screen } from '@testing-library/react'
import { DefinitionList } from './DefinitionList'

describe('DefinitionList', () => {
  it('項目名と値を dt・dd の組で並べる', () => {
    render(
      <DefinitionList
        items={[
          { term: '学籍番号', description: '26238001' },
          { term: '学部・学科', description: '理工学部 理工学科' },
        ]}
      />,
    )
    const terms = screen.getAllByRole('term')
    const definitions = screen.getAllByRole('definition')
    expect(terms.map((t) => t.textContent)).toEqual(['学籍番号', '学部・学科'])
    expect(definitions.map((d) => d.textContent)).toEqual(['26238001', '理工学部 理工学科'])
  })

  it('枠付き・小さい文字に切り替えられる', () => {
    const { container } = render(
      <DefinitionList
        items={[{ term: '有効期限', description: '2030年3月31日' }]}
        size="sm"
        boxed
      />,
    )
    const dl = container.querySelector('dl')
    expect(dl?.className).toMatch(/boxed/)
    expect(dl?.className).toMatch(/sm/)
  })
})
