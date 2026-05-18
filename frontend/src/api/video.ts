import api from './client'
import type { Video, PublishVideoRequest } from '../types'

export const videoAPI = {
  listByAuthorID: (author_id: number) => api.post<Video[]>('/video/listByAuthorID', { author_id }),
  getDetail: (id: number) => api.post<Video>('/video/getDetail', { id }),
  publish: (data: PublishVideoRequest) => api.post<Video>('/video/publish', data),
  uploadVideo: (file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    return api.post<{ url: string; play_url: string }>('/video/uploadVideo', fd, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 300000,
    })
  },
  uploadCover: (file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    return api.post<{ url: string; cover_url: string }>('/video/uploadCover', fd, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },
}
