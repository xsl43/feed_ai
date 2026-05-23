import api from './client'

export interface ReviewConfig {
  enabled: boolean
  text_model: string
  vision_model: string
  sample_frames: number
  frame_review_mode: string
  confidence_threshold: number
  manual_review_threshold: number
  max_retries: number
}

export interface ReviewConfigUpdate {
  enabled?: boolean
  text_model?: string
  vision_model?: string
  sample_frames?: number
  frame_review_mode?: string
  confidence_threshold?: number
  manual_review_threshold?: number
  max_retries?: number
}

export interface ReviewStatus {
  id: number
  review_status: string
  review_reason: string
  review_confidence?: number
}

export interface PendingVideo {
  id: number
  author_id: number
  username: string
  title: string
  description?: string
  cover_url: string
  play_url: string
  create_time: string
  likes_count: number
  popularity: number
  review_status: string
  review_reason?: string
  review_confidence?: number
  review_categories?: string
}

export const reviewAPI = {
  getConfig: () => api.get<ReviewConfig>('/review/config'),
  updateConfig: (cfg: ReviewConfigUpdate) => api.post('/review/config', cfg),
  getStatus: (videoId: number) => api.get<ReviewStatus>(`/review/status/${videoId}`),
  reSubmit: (id: number, title: string, description: string) =>
    api.post('/review/resubmit', { id, title, description }),
  getPending: () => api.get<PendingVideo[]>('/review/pending'),
  approveVideo: (id: number) => api.post(`/review/video/${id}/approve`),
  rejectVideo: (id: number, reason: string) =>
    api.post(`/review/video/${id}/reject`, { reason }),
}
