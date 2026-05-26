import { create } from 'zustand'
import type { Account } from '../types'

interface AuthState {
  token: string | null
  refreshToken: string | null
  user: Account | null
  isLoggedIn: boolean
  login: (token: string, refreshToken: string, user: Account) => void
  logout: () => void
  setUser: (user: Account) => void
  hydrate: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  token: null,
  refreshToken: null,
  user: null,
  isLoggedIn: false,

  login: (token, refreshToken, user) => {
    localStorage.setItem('token', token)
    localStorage.setItem('refresh_token', refreshToken)
    localStorage.setItem('user', JSON.stringify(user))
    set({ token, refreshToken, user, isLoggedIn: true })
  },

  logout: () => {
    localStorage.removeItem('token')
    localStorage.removeItem('refresh_token')
    localStorage.removeItem('user')
    set({ token: null, refreshToken: null, user: null, isLoggedIn: false })
  },

  setUser: (user) => set({ user }),

  hydrate: () => {
    const token = localStorage.getItem('token')
    const refreshToken = localStorage.getItem('refresh_token')
    const userStr = localStorage.getItem('user')
    if (token && userStr) {
      try {
        const user = JSON.parse(userStr) as Account
        set({ token, refreshToken, user, isLoggedIn: true })
      } catch {
        set({ token, refreshToken, isLoggedIn: true })
      }
    }
  },
}))
