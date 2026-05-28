// ========== Account ==========
export interface Account {
  id: number
  username: string
  avatar_url?: string
  bio?: string
  is_admin?: boolean
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  refresh_token: string
  account_id: number
  username: string
  is_admin: boolean
}

export interface RegisterRequest {
  username: string
  password: string
}

export interface RefreshRequest {
  refresh_token: string
}

export interface ChangePasswordRequest {
  username: string
  old_password: string
  new_password: string
}

export interface UpdateProfileRequest {
  avatar_url?: string
  bio?: string
}

export interface ProfileResponse {
  account: Account
  video_count: number
  total_likes: number
  follower_count: number
  vlogger_count: number
}
