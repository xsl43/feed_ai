import api from './client'
import type { Comment, PublishCommentRequest } from '../types'

export const commentAPI = {
  listAll: (video_id: number) => api.post<Comment[]>('/comment/listAll', { video_id }),
  publish: (data: PublishCommentRequest) => api.post('/comment/publish', data),
  delete: (comment_id: number) => api.post('/comment/delete', { comment_id }),
}

export const likeAPI = {
  like: (video_id: number) => api.post('/like/like', { video_id }),
  unlike: (video_id: number) => api.post('/like/unlike', { video_id }),
  isLiked: (video_id: number) => api.post<{ is_liked: boolean }>('/like/isLiked', { video_id }),
  listMyLikedVideos: () => api.post('/like/listMyLikedVideos'),
}
