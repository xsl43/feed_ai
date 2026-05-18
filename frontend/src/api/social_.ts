import api from './client'
import type { FollowersResponse, VloggersResponse, SocialCounts } from '../types'

export const socialAPI = {
  follow: (vlogger_id: number) => api.post('/social/follow', { vlogger_id }),
  unfollow: (vlogger_id: number) => api.post('/social/unfollow', { vlogger_id }),
  getAllFollowers: (vlogger_id?: number) =>
    api.post<FollowersResponse>('/social/getAllFollowers', { vlogger_id: vlogger_id || 0 }),
  getAllVloggers: (follower_id?: number) =>
    api.post<VloggersResponse>('/social/getAllVloggers', { follower_id: follower_id || 0 }),
  getCounts: () => api.post<SocialCounts>('/social/getCounts'),
}
