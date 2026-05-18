import axios from 'axios'

const api = axios.create({
  baseURL: '',
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' },
})

// 请求拦截器：自动附加 JWT Token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截器：自动刷新 Token
let isRefreshing = false
let refreshSubscribers: ((token: string) => void)[] = []

function onRefreshed(token: string) {
  refreshSubscribers.forEach((cb) => cb(token))
  refreshSubscribers = []
}

function addRefreshSubscriber(cb: (token: string) => void) {
  refreshSubscribers.push(cb)
}

api.interceptors.response.use(
  (res) => res,
  async (error) => {
    const original = error.config
    if (error.response?.status === 401 && !original._retry) {
      // Don't retry public endpoints — just clear token and proceed anonymously
      const publicEndpoints = ['/feed/', '/video/', '/comment/listAll', '/account/findByID', '/account/findByUsername', '/account/getProfile']
      const isPublic = publicEndpoints.some((ep) => original.url?.startsWith(ep))
      
      if (isPublic) {
        // Clear stale token and retry without auth
        localStorage.removeItem('token')
        delete original.headers.Authorization
        original._retry = true
        return api(original)
      }

      original._retry = true
      const refreshToken = localStorage.getItem('refresh_token')
      if (!refreshToken) {
        localStorage.removeItem('token')
        return Promise.reject(error)
      }

      if (!isRefreshing) {
        isRefreshing = true
        try {
          const { data } = await axios.post('/account/refresh', { refresh_token: refreshToken })
          localStorage.setItem('token', data.token)
          api.defaults.headers.common.Authorization = `Bearer ${data.token}`
          onRefreshed(data.token)
          original.headers.Authorization = `Bearer ${data.token}`
          return api(original)
        } catch {
          localStorage.removeItem('token')
          localStorage.removeItem('refresh_token')
          isRefreshing = false
          window.location.href = '/login'
          return Promise.reject(error)
        } finally {
          isRefreshing = false
        }
      }

      // Queue other requests while refreshing
      return new Promise((resolve) => {
        addRefreshSubscriber((token: string) => {
          original.headers.Authorization = `Bearer ${token}`
          resolve(api(original))
        })
      })
    }
    return Promise.reject(error)
  }
)

export default api
