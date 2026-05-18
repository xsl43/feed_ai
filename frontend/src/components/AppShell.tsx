import { Outlet, Link, useNavigate } from 'react-router-dom'
import { useAuthStore } from '../store/authStore'
import { useEffect } from 'react'

export default function AppShell() {
  const { isLoggedIn, user, hydrate, logout } = useAuthStore()
  const navigate = useNavigate()

  useEffect(() => { hydrate() }, [])

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  return (
    <div className="min-h-screen bg-weibo-bg">
      {/* 顶部导航 */}
      <header className="sticky top-0 z-50 bg-white/95 backdrop-blur-sm border-b border-weibo-border">
        <div className="max-w-6xl mx-auto px-4 h-14 flex items-center justify-between">
          <Link to="/" className="flex items-center gap-2">
            <span className="text-weibo-primary text-xl font-bold">FeedAI</span>
            <span className="text-weibo-text-secondary text-xs hidden sm:inline">智能短视频平台</span>
          </Link>

          <nav className="flex items-center gap-1 sm:gap-2">
            <NavLink to="/" label="首页" />
            <NavLink to="/ai" label="AI分析" />
            {isLoggedIn ? (
              <>
                <NavLink to="/publish" label="发布" />
                <NavLink to="/messages" label="消息" />
                <Link
                  to={`/profile/${user?.id || 1}`}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-full hover:bg-weibo-bg transition-colors text-sm"
                >
                  <div className="w-6 h-6 rounded-full bg-weibo-primary text-white flex items-center justify-center text-xs font-medium">
                    {(user?.username || 'U')[0].toUpperCase()}
                  </div>
                  <span className="hidden sm:inline text-weibo-text">{user?.username}</span>
                </Link>
                <button
                  onClick={handleLogout}
                  className="text-weibo-text-secondary text-sm hover:text-weibo-primary px-2 py-1"
                >
                  退出
                </button>
              </>
            ) : (
              <Link to="/login" className="btn-primary text-sm !px-4 !py-1.5">
                登录
              </Link>
            )}
          </nav>
        </div>
      </header>

      {/* 主内容 */}
      <main className="max-w-6xl mx-auto px-4 py-4">
        <Outlet />
      </main>
    </div>
  )
}

function NavLink({ to, label }: { to: string; label: string }) {
  return (
    <Link
      to={to}
      className="px-3 py-1.5 rounded-full text-sm text-weibo-text-secondary hover:text-weibo-text hover:bg-weibo-bg transition-colors"
    >
      {label}
    </Link>
  )
}
