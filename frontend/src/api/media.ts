import api from './client'
import type { MediaFile } from '../types'

export const mediaAPI = {
  initUpload: () => api.post<{ upload_id: string }>('/media/init-upload'),
  upload: (file: File, user_id?: number) => {
    const fd = new FormData()
    fd.append('file', file)
    if (user_id) fd.append('user_id', String(user_id))
    return api.post<{ message: string; id: number; url: string }>('/media/upload', fd, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 300000,
    })
  },
  uploadChunk: (upload_id: string, chunk_index: number, chunk: Blob) => {
    const fd = new FormData()
    fd.append('upload_id', upload_id)
    fd.append('chunk_index', String(chunk_index))
    fd.append('chunk', chunk)
    return api.post('/media/upload-chunk', fd, { headers: { 'Content-Type': 'multipart/form-data' } })
  },
  completeUpload: (upload_id: string, filename: string, md5?: string, user_id?: number) => {
    const fd = new FormData()
    fd.append('upload_id', upload_id)
    fd.append('filename', filename)
    if (md5) fd.append('md5', md5)
    if (user_id) fd.append('user_id', String(user_id))
    return api.post<{ message: string; id: number; url: string }>('/media/complete-upload', fd, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 300000,
    })
  },
  list: (user_id?: number) => api.get<MediaFile[]>('/media/list', { params: { user_id } }),
  delete: (id: number, user_id: number) =>
    api.delete('/media/delete', { params: { id, user_id } }),
  checkDuplicate: (md5: string) => api.post<{ duplicate: boolean }>('/media/check-duplicate', { md5 }),
}
