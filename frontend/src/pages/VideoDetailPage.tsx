import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { videoAPI, reviewAPI } from '../api'
import type { Video } from '../types'
import VideoPlayer from '../components/VideoPlayer'
import CommentDrawer from '../components/CommentDrawer'
import { LikeButton, FollowButton } from '../components/Buttons'
import { useAuthStore } from '../store/authStore'
import { formatDate, formatCount } from '../utils/time'
import toast from 'react-hot-toast'

export default function VideoDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [video, setVideo] = useState<Video | null>(null)
  const [loading, setLoading] = useState(true)
  const [showComments, setShowComments] = useState(false)
  const { user } = useAuthStore()

  useEffect(() => {
    if (!id) return
    videoAPI.getDetail(Number(id)).then(({ data }) => {
      setVideo(data)
    }).catch(() => {}).finally(() => setLoading(false))
  }, [id])

  if (loading) {
    return <div className="flex justify-center py-16 text-weibo-text-muted">加载中...</div>
  }

  if (!video) {
    return <div className="text-center py-16 text-weibo-text-muted">视频不存在</div>
  }

  const isOwner = user?.id === video.author_id

  const handleReSubmit = async () => {
    try {
      await reviewAPI.reSubmit(video.id, video.title, video.description || '')
      toast.success('已重新提交审核')
      video.review_status = 'pending'
      video.review_reason = ''
      setVideo({ ...video })
    } catch (err: any) {
      toast.error(err.response?.data?.error || '重新提交失败')
    }
  }

  return (
    <div className="max-w-4xl mx-auto">
      {/* 审核状态提示 (仅作者可见) */}
      {isOwner && video.review_status && video.review_status !== 'approved' && (
        <div className={`mb-3 p-3 rounded-lg text-sm ${
          video.review_status === 'pending' ? 'bg-yellow-50 text-yellow-700 border border-yellow-200' :
          'bg-red-50 text-red-700 border border-red-200'
        }`}>
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium">
                {video.review_status === 'pending' ? '审核中' : '审核未通过'}
              </p>
              {video.review_reason && (
                <p className="text-xs mt-0.5">原因: {video.review_reason}</p>
              )}
            </div>
            {video.review_status === 'rejected' && (
              <button onClick={handleReSubmit} className="text-xs bg-red-100 hover:bg-red-200 px-3 py-1 rounded-full font-medium transition-colors">
                重新提交
              </button>
            )}
          </div>
        </div>
      )}

      {/* 视频播放器 */}
      <VideoPlayer src={video.play_url} poster={video.cover_url} className="mb-4" />

      {/* 视频信息 */}
      <div className="card">
        <h1 className="text-lg font-semibold text-weibo-text mb-3">{video.title}</h1>

        <div className="flex items-center justify-between flex-wrap gap-3">
          {/* 作者信息 */}
          <Link to={`/profile/${video.author_id}`} className="flex items-center gap-3 group">
            <div className="w-10 h-10 rounded-full bg-gradient-to-br from-weibo-primary to-weibo-link text-white flex items-center justify-center text-sm font-bold">
              {video.username[0]?.toUpperCase() || 'U'}
            </div>
            <div>
              <p className="text-sm font-medium text-weibo-text group-hover:text-weibo-primary transition-colors">
                {video.username}
              </p>
              <p className="text-2xs text-weibo-text-muted">{formatDate(video.create_time)}</p>
            </div>
          </Link>

          {/* 操作按钮 */}
          <div className="flex items-center gap-2">
            <FollowButton vloggerId={video.author_id} />
            <LikeButton videoId={video.id} initialLiked={false} />
            <button
              onClick={() => setShowComments(true)}
              className="flex items-center gap-1.5 px-4 py-2 rounded-full border border-weibo-border-strong text-weibo-text-secondary hover:border-weibo-primary hover:text-weibo-primary transition-all text-sm font-medium"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
              </svg>
              评论
            </button>
          </div>
        </div>

        {/* 描述 & 数据 */}
        {video.description && (
          <p className="mt-3 text-sm text-weibo-text-secondary leading-relaxed">{video.description}</p>
        )}
        <div className="flex items-center gap-4 mt-3 text-xs text-weibo-text-muted">
          <span>👍 {formatCount(video.likes_count)} 赞</span>
          <span>🔥 热度 {formatCount(video.popularity)}</span>
        </div>
      </div>

      {/* 评论抽屉 */}
      <CommentDrawer
        videoId={video.id}
        open={showComments}
        onClose={() => setShowComments(false)}
      />
    </div>
  )
}
