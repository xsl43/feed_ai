import { useState, useEffect } from 'react'
import { messageAPI, accountAPI } from '../api'
import type { Message } from '../types'
import { useAuthStore } from '../store/authStore'
import { timeAgo } from '../utils/time'
import toast from 'react-hot-toast'

export default function MessagesPage() {
  const [messages, setMessages] = useState<Message[]>([])
  const [peerId, setPeerId] = useState('')
  const [content, setContent] = useState('')
  const [loading, setLoading] = useState(false)
  const [sending, setSending] = useState(false)
  const [peerUsername, setPeerUsername] = useState('')
  const { user, isLoggedIn } = useAuthStore()

  const loadMessages = async () => {
    const pid = Number(peerId)
    if (!pid) return
    setLoading(true)
    try {
      const { data } = await messageAPI.list(pid)
      setMessages(data.messages || [])
      const accRes = await accountAPI.findByID(pid)
      setPeerUsername(accRes.data.username)
    } catch {
      toast.error('加载消息失败')
    } finally {
      setLoading(false)
    }
  }

  const handleSend = async () => {
    if (!content.trim() || !peerId) return
    setSending(true)
    try {
      await messageAPI.send({ to_id: Number(peerId), content: content.trim() })
      toast.success('发送成功')
      setContent('')
      loadMessages()
    } catch {
      toast.error('发送失败')
    } finally {
      setSending(false)
    }
  }

  if (!isLoggedIn) {
    return (
      <div className="text-center py-16 text-weibo-text-muted">
        请先登录后查看消息
      </div>
    )
  }

  return (
    <div className="max-w-2xl mx-auto">
      <h1 className="text-xl font-bold text-weibo-text mb-4">私信</h1>

      <div className="card mb-4">
        <div className="flex gap-2">
          <input
            type="number"
            value={peerId}
            onChange={(e) => setPeerId(e.target.value)}
            className="input-field flex-1"
            placeholder="输入对方用户ID"
          />
          <button onClick={loadMessages} disabled={!peerId} className="btn-primary text-sm">
            加载
          </button>
        </div>
      </div>

      {peerUsername && (
        <div className="mb-3 text-sm text-weibo-text-secondary">
          与 <span className="font-medium text-weibo-text">{peerUsername}</span> 的对话
        </div>
      )}

      {/* 消息列表 */}
      <div className="card mb-4 max-h-96 overflow-y-auto space-y-3">
        {loading ? (
          <div className="text-center py-8 text-weibo-text-muted text-sm">加载中...</div>
        ) : messages.length === 0 ? (
          <div className="text-center py-8 text-weibo-text-muted text-sm">
            {peerId ? '暂无消息' : '输入用户ID后加载消息'}
          </div>
        ) : (
          messages.map((m) => {
            const isMine = m.from_id === user?.id
            return (
              <div key={m.id} className={`flex ${isMine ? 'justify-end' : 'justify-start'}`}>
                <div className={`max-w-[75%] rounded-2xl px-4 py-2.5 text-sm ${
                  isMine
                    ? 'bg-weibo-primary text-white rounded-br-md'
                    : 'bg-weibo-bg text-weibo-text rounded-bl-md'
                }`}>
                  <p>{m.content}</p>
                  <p className={`text-2xs mt-1 ${isMine ? 'text-white/70' : 'text-weibo-text-muted'}`}>
                    {timeAgo(m.created_at)}
                  </p>
                </div>
              </div>
            )
          })
        )}
      </div>

      {/* 输入区 */}
      {peerId && (
        <div className="card">
          <div className="flex gap-2">
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              onKeyDown={(e) => { if (e.ctrlKey && e.key === 'Enter') handleSend() }}
              className="input-field resize-none flex-1"
              rows={2}
              placeholder="输入消息..."
            />
            <button
              onClick={handleSend}
              disabled={!content.trim() || sending}
              className="self-end btn-primary text-sm !px-6"
            >
              {sending ? '...' : '发送'}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
