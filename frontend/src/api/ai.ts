import api from './client'
import type { AIStatus } from '../types'

export interface AIConfig {
  api_key: string
  base_url: string
  model: string
  asr_model: string
}

export interface AIConfigUpdate {
  api_key?: string
  base_url?: string
  model?: string
  asr_model?: string
}

export const aiAPI = {
  analyze: (media_id: number) => api.post('/ai/analyze', { media_id }),
  transcribe: (media_id: number) => api.post('/ai/transcribe', { media_id }),
  summarize: (text: string) => api.post<{ summary: string }>('/ai/summarize', { text }),
  getStatus: (id: number) => api.get<AIStatus>(`/ai/status/${id}`),
  downloadAudio: (id: number) => api.get(`/ai/audio/${id}`, { responseType: 'blob' }),
  getConfig: () => api.get<AIConfig>('/ai/config'),
  updateConfig: (cfg: AIConfigUpdate) =>
    api.post<{ message: string; config: Partial<AIConfig> }>('/ai/config', cfg),
}
