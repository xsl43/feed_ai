import api from './client'
import type { FeedVideoItem, ListLatestResponse, ListPopularityResponse, ListLikesCountResponse } from '../types'

export const feedAPI = {
  listLatest: (limit = 10, latest_time = 0) =>
    api.post<ListLatestResponse>('/feed/listLatest', { limit, latest_time }),
  listLikesCount: (limit = 10, cursor?: { likes_count_before?: number; id_before?: number }) =>
    api.post<ListLikesCountResponse>('/feed/listLikesCount', { limit, ...cursor }),
  listByPopularity: (limit = 10, as_of = 0, offset = 0) =>
    api.post<ListPopularityResponse>('/feed/listByPopularity', { limit, as_of, offset }),
  listByFollowing: (limit = 10, latest_time = 0) =>
    api.post<{ video_list: FeedVideoItem[]; next_time: number; has_more: boolean }>(
      '/feed/listByFollowing',
      { limit, latest_time }
    ),
  listByTag: (tag_name: string, limit = 10) =>
    api.post<{ video_list: FeedVideoItem[] }>('/feed/listByTag', { tag_name, limit }),
}
