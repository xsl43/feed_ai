import { useState } from 'react'
import { likeAPI, socialAPI } from '../api'
import { useAuthStore } from '../store/authStore'
import toast from 'react-hot-toast'

interface LikeButtonProps {
  videoId: number
  initialLiked: boolean
  className?: string
}

export function LikeButton({ videoId, initialLiked, className = '' }: LikeButtonProps) {
  const [liked, setLiked] = useState(initialLiked)
  const [loading, setLoading] = useState(false)
  const { isLoggedIn } = useAuthStore()

  const toggle = async () => {
    if (!isLoggedIn) { toast.error('请先登录'); return }
    if (loading) return
    setLoading(true)
    try {
      if (liked) {
        await likeAPI.unlike(videoId)
        setLiked(false)
      } else {
        await likeAPI.like(videoId)
        setLiked(true)
      }
    } catch {
      toast.error('操作失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <button
      onClick={toggle}
      disabled={loading}
      className={`flex items-center gap-1.5 px-4 py-2 rounded-full border transition-all duration-200 text-sm font-medium ${
        liked
          ? 'border-weibo-primary bg-weibo-highlight text-weibo-primary'
          : 'border-weibo-border-strong text-weibo-text-secondary hover:border-weibo-primary hover:text-weibo-primary'
      } ${className}`}
    >
      <svg className="w-4 h-4" fill={liked ? '#E6162D' : 'none'} stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
        <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z" />
      </svg>
      {liked ? '已赞' : '点赞'}
    </button>
  )
}

interface FollowButtonProps {
  vloggerId: number
  className?: string
}

export function FollowButton({ vloggerId, className = '' }: FollowButtonProps) {
  const [following, setFollowing] = useState(false)
  const [loading, setLoading] = useState(false)
  const { isLoggedIn } = useAuthStore()

  const toggle = async () => {
    if (!isLoggedIn) { toast.error('请先登录'); return }
    if (loading) return
    setLoading(true)
    try {
      if (following) {
        await socialAPI.unfollow(vloggerId)
        setFollowing(false)
        toast.success('已取消关注')
      } else {
        await socialAPI.follow(vloggerId)
        setFollowing(true)
        toast.success('关注成功')
      }
    } catch {
      toast.error('操作失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <button
      onClick={toggle}
      disabled={loading}
      className={`px-5 py-1.5 rounded-full text-sm font-medium transition-all duration-200 ${
        following
          ? 'border border-weibo-border-strong text-weibo-text-secondary bg-weibo-bg hover:bg-weibo-border'
          : 'bg-weibo-primary text-white hover:bg-weibo-primary-hover'
      } ${className}`}
    >
      {following ? '已关注' : '+ 关注'}
    </button>
  )
}
