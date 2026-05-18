export interface SocialCounts {
  follower_count: number
  vlogger_count: number
}

export interface FollowersResponse {
  followers: { id: number; username: string; avatar_url?: string; bio?: string }[]
  follower_count: number
}

export interface VloggersResponse {
  vloggers: { id: number; username: string; avatar_url?: string; bio?: string }[]
  vlogger_count: number
}
