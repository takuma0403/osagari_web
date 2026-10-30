// 利用規約・プライバシーポリシーの本文。vite/legalMarkdown.ts が docs/legal の正本をビルド時に HTML にする
declare module 'virtual:legal/*' {
  const html: string
  export default html
}
