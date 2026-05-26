import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { accountAPI, videoAPI } from '../api'
import type { ProfileResponse, Video } from '../types'
import { FollowButton } from '../components/Buttons'
import { formatCount } from '../utils/time'
import { useAuthStore } from '../store/authStore'
import toast from 'react-hot-toast'

export default function ProfilePage() {
  const { id } = useParams<{ id: string }>()
  const [profile, setProfile] = useState<ProfileResponse | null>(null)
  const [videos, setVideos] = useState<Video[]>([])
  const [loading, setLoading] = useState(true)
  const { user } = useAuthStore()

  const isSelf = user?.id === Number(id)

  const handleDelete = async (videoId: number) => {
    if (!confirm('确定要删除这个视频吗？')) return
    try {
      await videoAPI.delete(videoId)
      toast.success('已删除')
      setVideos((prev) => prev.filter((v) => v.id !== videoId))
    } catch (err: any) {
      toast.error(err.response?.data?.error || '删除失败')
    }
  }

  useEffect(() => {
    if (!id) return
    const numId = Number(id)
    Promise.all([
      accountAPI.getProfile(numId),
      videoAPI.listByAuthorID(numId),
    ]).then(([pRes, vRes]) => {
      setProfile(pRes.data)
      setVideos(vRes.data || [])
    }).finally(() => setLoading(false))
  }, [id])

  if (loading) {
    return <div className="flex justify-center py-16 text-weibo-text-muted">加载中...</div>
  }

  if (!profile) {
    return <div className="text-center py-16 text-weibo-text-muted">用户不存在</div>
  }

  return (
    <div className="max-w-2xl mx-auto">
      {/* 个人信息卡片 */}
      <div className="card mb-4">
        <div className="flex items-start gap-4">
          <div className="w-20 h-20 rounded-full bg-gradient-to-br from-weibo-primary to-weibo-link text-white flex items-center justify-center text-2xl font-bold shrink-0">
            {profile.account.username[0]?.toUpperCase()}
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-3 mb-1">
              <h1 className="text-xl font-bold text-weibo-text">{profile.account.username}</h1>
              {!isSelf && <FollowButton vloggerId={profile.account.id} />}
            </div>
            {profile.account.bio && (
              <p className="text-sm text-weibo-text-secondary mb-3">{profile.account.bio}</p>
            )}
            <div className="flex items-center gap-5 text-sm">
              <span><strong className="text-weibo-text">{videos.length}</strong> <span className="text-weibo-text-muted">视频</span></span>
              <span><strong className="text-weibo-text">{formatCount(profile.total_likes)}</strong> <span className="text-weibo-text-muted">获赞</span></span>
              <span><strong className="text-weibo-text">{formatCount(profile.follower_count)}</strong> <span className="text-weibo-text-muted">粉丝</span></span>
              <span><strong className="text-weibo-text">{formatCount(profile.vlogger_count)}</strong> <span className="text-weibo-text-muted">关注</span></span>
            </div>
          </div>
        </div>
      </div>

      {/* 视频网格 */}
      <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
        {videos.length === 0 ? (
          <div className="col-span-full text-center py-12 text-weibo-text-muted text-sm">
            暂无作品
          </div>
        ) : (
          videos.map((v) => (
            <div key={v.id} className="group relative rounded-lg overflow-hidden bg-weibo-bg aspect-video">
              <Link to={`/video/${v.id}`}>
                <img
                  src={v.cover_url}
                  alt={v.title}
                  className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                  loading="lazy"
                />
                <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/70 to-transparent p-2 flex justify-between items-end">
                  <p className="text-white text-xs line-clamp-1 flex-1">{v.title}</p>
                  <span className="text-white text-2xs ml-2 shrink-0">👍 {formatCount(v.likes_count)}</span>
                </div>
              </Link>
              {isSelf && (
                <button
                  onClick={(e) => { e.preventDefault(); handleDelete(v.id) }}
                  className="absolute top-1 left-1 bg-red-500/80 hover:bg-red-600 text-white text-2xs px-1.5 py-0.5 rounded"
                >
                  删除
                </button>
              )}
              {isSelf && v.review_status && v.review_status !== 'approved' && (
                <div className={`absolute top-1 right-1 text-2xs px-1.5 py-0.5 rounded ${
                  v.review_status === 'pending' ? 'bg-yellow-400 text-yellow-900' : 'bg-red-500 text-white'
                }`}>
                  {v.review_status === 'pending' ? '审核中' : '未通过'}
                </div>
              )}
            </div>
          ))
        )}
      </div>
    </div>
  )
}
