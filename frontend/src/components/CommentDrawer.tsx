import { useState, useEffect, useRef } from 'react'
import { commentAPI } from '../api'
import type { Comment } from '../types'
import { useAuthStore } from '../store/authStore'
import { timeAgo } from '../utils/time'
import toast from 'react-hot-toast'

interface Props {
  videoId: number
  open: boolean
  onClose: () => void
}

export default function CommentDrawer({ videoId, open, onClose }: Props) {
  const [comments, setComments] = useState<Comment[]>([])
  const [content, setContent] = useState('')
  const [loading, setLoading] = useState(false)
  const { isLoggedIn } = useAuthStore()
  const inputRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    if (open && videoId) {
      loadComments()
    }
  }, [open, videoId])

  const loadComments = async () => {
    setLoading(true)
    try {
      const { data } = await commentAPI.listAll(videoId)
      setComments(data || [])
    } catch {
      toast.error('加载评论失败')
    } finally {
      setLoading(false)
    }
  }

  const handlePublish = async () => {
    if (!content.trim()) return
    if (!isLoggedIn) {
      toast.error('请先登录')
      return
    }
    try {
      await commentAPI.publish({ video_id: videoId, content: content.trim() })
      toast.success('评论成功')
      setContent('')
      loadComments()
    } catch {
      toast.error('评论失败')
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.ctrlKey && e.key === 'Enter') {
      handlePublish()
    }
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div className="absolute inset-0 bg-black/30" onClick={onClose} />
      <div className="relative w-full max-w-md bg-white h-full shadow-xl flex flex-col animate-slide-in">
        {/* 头部 */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-weibo-border">
          <h3 className="font-semibold text-weibo-text">评论</h3>
          <button onClick={onClose} className="text-weibo-text-muted hover:text-weibo-text p-1">
            <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
              <path d="M18 6L6 18M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* 评论列表 */}
        <div className="flex-1 overflow-y-auto px-4 py-3 space-y-3">
          {loading ? (
            <div className="flex justify-center py-8 text-weibo-text-muted text-sm">加载中...</div>
          ) : comments.length === 0 ? (
            <div className="text-center py-8 text-weibo-text-muted text-sm">暂无评论，快来抢沙发吧~</div>
          ) : (
            comments.map((c) => (
              <div key={c.id} className="flex gap-2.5">
                <div className="w-7 h-7 rounded-full bg-weibo-border-strong flex items-center justify-center text-xs font-bold text-weibo-text-secondary shrink-0">
                  {c.username[0]?.toUpperCase()}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-baseline gap-2">
                    <span className="text-xs font-medium text-weibo-link">{c.username}</span>
                    <span className="text-2xs text-weibo-text-muted">{timeAgo(c.created_at)}</span>
                  </div>
                  <p className="text-sm text-weibo-text mt-0.5 break-words">{c.content}</p>
                </div>
              </div>
            ))
          )}
        </div>

        {/* 输入区 */}
        <div className="border-t border-weibo-border p-3">
          <div className="flex gap-2">
            <textarea
              ref={inputRef}
              value={content}
              onChange={(e) => setContent(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={isLoggedIn ? '说点什么...' : '请先登录'}
              disabled={!isLoggedIn}
              className="flex-1 resize-none rounded-lg border border-weibo-border bg-weibo-bg px-3 py-2 text-sm focus:outline-none focus:border-weibo-primary disabled:opacity-50"
              rows={2}
            />
            <button
              onClick={handlePublish}
              disabled={!content.trim() || !isLoggedIn}
              className="self-end btn-primary text-sm !px-4 !py-2"
            >
              发送
            </button>
          </div>
          <p className="text-2xs text-weibo-text-muted mt-1.5">Ctrl + Enter 快捷发送</p>
        </div>
      </div>
    </div>
  )
}
