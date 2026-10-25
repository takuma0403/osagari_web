import { useEffect, useRef, useState } from 'react'
import type { ScanControls, StartScan } from './cameraScan'

type Options = {
  /** true の間だけカメラを使う。false にしたら止める */
  active: boolean
  startScan: StartScan
  /** 読み取った値を受け取る。true を返したら読み取りを止める */
  onText: (text: string) => boolean | void
}

/**
 * カメラの読み取りを active の間だけ動かす（FE）。起動を待っている間に止めた場合や、画面を離れた場合もカメラを止める。
 * 返す videoRef を <video> に付ける。カメラを起動できなかったときは error にその理由（getUserMedia のエラーなど）が入る。
 */
export function useCameraScan({ active, startScan, onText }: Options) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const [error, setError] = useState<unknown>(null)
  const onTextRef = useRef(onText)
  useEffect(() => {
    onTextRef.current = onText
  }, [onText])

  useEffect(() => {
    const video = videoRef.current
    if (!active || !video) return
    let controls: ScanControls | undefined
    let cancelled = false
    setError(null)
    startScan(video, (text) => {
      if (cancelled) return
      if (onTextRef.current(text) === true) {
        cancelled = true
        controls?.stop()
      }
    })
      .then((c) => {
        // 起動を待っている間に止めたら、すぐ止める
        if (cancelled) c.stop()
        else controls = c
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err ?? new Error('camera'))
      })
    return () => {
      cancelled = true
      controls?.stop()
    }
  }, [active, startScan])

  return { videoRef, error }
}

/** カメラの使用が許可されていないエラーか */
export function isCameraDenied(err: unknown): boolean {
  return err instanceof DOMException && err.name === 'NotAllowedError'
}
