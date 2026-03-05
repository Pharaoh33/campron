import { http } from './client'
import type { DownloadReq, DownloadResp } from './types'

export async function downloadPronunciation(payload: DownloadReq): Promise<DownloadResp> {
  const { data } = await http.post<DownloadResp>('/api/v1/pronunciations/download', payload)
  return data
}
