import api from './client'
import type { LoginRequest, LoginResponse, RegisterRequest, Account, ProfileResponse, UpdateProfileRequest } from '../types'

export const accountAPI = {
  register: (data: RegisterRequest) => api.post<{ message: string }>('/account/register', data),
  login: (data: LoginRequest) => api.post<LoginResponse>('/account/login', data),
  findByID: (id: number) => api.post<Account>('/account/findByID', { id }),
  findByUsername: (username: string) => api.post<Account>('/account/findByUsername', { username }),
  getProfile: (account_id: number) => api.post<ProfileResponse>('/account/getProfile', { account_id }),
  changePassword: (data: { username: string; old_password: string; new_password: string }) =>
    api.post('/account/changePassword', data),
  refresh: (refresh_token: string) => api.post<LoginResponse>('/account/refresh', { refresh_token }),
  logout: () => api.post('/account/logout'),
  rename: (new_username: string) => api.post('/account/rename', { new_username }),
  uploadAvatar: (file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    return api.post<{ avatar_url: string }>('/account/uploadAvatar', fd, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },
  updateProfile: (data: UpdateProfileRequest) => api.post('/account/updateProfile', data),
}
