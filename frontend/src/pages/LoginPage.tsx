import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { accountAPI } from '../api'
import { useAuthStore } from '../store/authStore'
import toast from 'react-hot-toast'

export default function LoginPage() {
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()
  const { login } = useAuthStore()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!username.trim() || !password.trim()) {
      toast.error('请填写完整信息')
      return
    }
    setLoading(true)
    try {
      if (mode === 'register') {
        await accountAPI.register({ username: username.trim(), password })
        toast.success('注册成功，请登录')
        setMode('login')
        setPassword('')
      } else {
        const { data } = await accountAPI.login({ username: username.trim(), password })
        login(data.token, data.refresh_token, { id: data.account_id, username: data.username })
        toast.success(`欢迎回来，${data.username}!`)
        navigate('/')
      }
    } catch (err: any) {
      toast.error(err.response?.data?.error || '操作失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-weibo-bg flex items-center justify-center p-4">
      <div className="w-full max-w-sm">
        {/* Logo */}
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-weibo-primary mb-1">FeedAI</h1>
          <p className="text-weibo-text-secondary text-sm">智能短视频平台</p>
        </div>

        {/* 表单卡片 */}
        <div className="card">
          <div className="flex mb-6 bg-weibo-bg rounded-lg p-1">
            <button
              onClick={() => setMode('login')}
              className={`flex-1 py-2 rounded-md text-sm font-medium transition-colors ${
                mode === 'login' ? 'bg-white text-weibo-text shadow-sm' : 'text-weibo-text-secondary'
              }`}
            >
              登录
            </button>
            <button
              onClick={() => setMode('register')}
              className={`flex-1 py-2 rounded-md text-sm font-medium transition-colors ${
                mode === 'register' ? 'bg-white text-weibo-text shadow-sm' : 'text-weibo-text-secondary'
              }`}
            >
              注册
            </button>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm text-weibo-text-secondary mb-1">用户名</label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="input-field"
                placeholder="请输入用户名"
                autoComplete="username"
              />
            </div>
            <div>
              <label className="block text-sm text-weibo-text-secondary mb-1">密码</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="input-field"
                placeholder="请输入密码"
                autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
              />
            </div>
            <button
              type="submit"
              disabled={loading}
              className="btn-primary w-full !py-2.5"
            >
              {loading ? '请稍候...' : mode === 'login' ? '登录' : '注册'}
            </button>
          </form>
        </div>
      </div>
    </div>
  )
}
