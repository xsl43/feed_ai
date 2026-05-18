export interface MediaFile {
  id: number
  user_id?: number
  filename: string
  file_path: string
  file_size: number
  status: string
  ai_summary?: string
  transcript_text?: string
  cover_url?: string
  upload_time: string
  created_at: string
}

export interface AIStatus {
  id: number
  status: string
  ai_summary?: string
  transcript_text?: string
}
