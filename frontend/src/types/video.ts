export interface Video {
  id: number
  author_id: number
  username: string
  title: string
  description?: string
  play_url: string
  cover_url: string
  create_time: string
  likes_count: number
  popularity: number
}

export interface FeedVideoItem {
  id: number
  author: { id: number; username: string }
  title: string
  description?: string
  play_url: string
  cover_url: string
  create_time: number
  likes_count: number
  is_liked: boolean
}

export interface PublishVideoRequest {
  title: string
  description: string
  play_url: string
  cover_url: string
}

export interface ListLatestResponse {
  video_list: FeedVideoItem[]
  next_time: number
  has_more: boolean
}

export interface ListPopularityResponse {
  video_list: FeedVideoItem[]
  as_of: number
  next_offset: number
  has_more: boolean
}

export interface ListLikesCountResponse {
  video_list: FeedVideoItem[]
  next_likes_count_before?: number
  next_id_before?: number
  has_more: boolean
}
