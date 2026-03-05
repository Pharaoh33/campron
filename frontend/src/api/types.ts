export type Accent = 'us' | 'uk' | 'both'

export interface DownloadReq {
  word: string
  accent: Accent
}

export interface SavedItem {
  accent: string
  folder: string
  mp3_filename: string
  mp3_path: string
  mp3_url: string
  ipa: string
  ipa_filename: string
  ipa_path: string
}

export interface DownloadResp {
  ok: boolean
  message?: string
  page_url?: string
  saved?: SavedItem[]
}
