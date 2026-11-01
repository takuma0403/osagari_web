import { cameraScanner } from '../../../../lib/cameraScan'

export type { ScanControls, StartScan } from '../../../../lib/cameraScan'

/** 背面カメラで EAN-13（書籍の ISBN バーコード）を読み取る（ITEM） */
export const startIsbnScan = cameraScanner(['EAN_13'])
