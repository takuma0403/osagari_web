import { ApiError } from '../api/client'

/** API が発行した署名付きURLのアップロード先（STORAGE。学生証・商品写真で共通の形） */
export type SignedUploadTarget = {
  uploadUrl: string
  method: string
  /** 署名に含まれるヘッダー。1つでも欠けると Cloud Storage が受け付けない */
  headers: Record<string, string>
}

const uploadFailedMessage = '画像をアップロードできませんでした。もう一度お試しください'

/**
 * 署名付きURLへ画像を直接送る（ARCH, STORAGE。API を経由しない）。
 * 発行時の headers をすべてそのまま付ける。宛先は Cloud Storage なので、API 用の request（Cookie・X-Device-Id）は使わない。
 * 失敗は ApiError で投げる（2xx 以外は upload_failed、通信の失敗は network_error）。
 * 同じURLには一度しか送れないため、やり直すときは URL を発行し直す。
 */
export async function uploadToSignedUrl(target: SignedUploadTarget, body: Blob): Promise<void> {
  let res: Response
  try {
    res = await fetch(target.uploadUrl, {
      method: target.method,
      headers: target.headers,
      body,
      credentials: 'omit',
    })
  } catch {
    throw ApiError.network()
  }
  if (!res.ok) throw new ApiError(res.status, 'upload_failed', uploadFailedMessage)
}
