import { useEffect, useState } from 'react'
import { request } from './api/client'
import type { components } from './api/schema.gen'

type Health = components['schemas']['Health']

// 環境構築の疎通確認用の仮画面。画面の実装（design-to-code）で置き換える
function App() {
  const [status, setStatus] = useState<string>('確認中…')

  useEffect(() => {
    request<Health>('/health')
      .then((h) => setStatus(h.status))
      .catch(() => setStatus('API に接続できません'))
  }, [])

  return (
    <main>
      <h1>佐賀大お下がりフリマ</h1>
      <p>API: {status}</p>
    </main>
  )
}

export default App
