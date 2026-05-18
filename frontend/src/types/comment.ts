export interface Comment {
  id: number
  username: string
  video_id: number
  author_id: number
  content: string
  created_at: string
}

export interface PublishCommentRequest {
  video_id: number
  content: string
}
