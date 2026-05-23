import { useState, useEffect } from 'react'
import { reviewAPI } from '../api'
import type { PendingVideo, ReviewConfig } from '../api'
import { formatDate } from '../utils/time'
import toast from 'react-hot-toast'

export default function AdminReviewPage() {
  const [videos, setVideos] = useState<PendingVideo[]>([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<'queue' | 'config'>('queue')
  const [viewVideoId, setViewVideoId] = useState<number | null>(null)
  const [rejectModal, setRejectModal] = useState<{ id: number; reason: string } | null>(null)

  // Config state
  const [cfg, setCfg] = useState<ReviewConfig | null>(null)
  const [cfgSaving, setCfgSaving] = useState(false)

  const loadVideos = async () => {
    try {
      const { data } = await reviewAPI.getPending()
      setVideos(data || [])
    } catch {
      toast.error('加载待审核列表失败')
    } finally {
      setLoading(false)
    }
  }

  const loadConfig = async () => {
    try {
      const { data } = await reviewAPI.getConfig()
      setCfg(data)
    } catch { /* ignore */ }
  }

  useEffect(() => {
    loadVideos()
    loadConfig()
  }, [])

  const handleApprove = async (id: number) => {
    try {
      await reviewAPI.approveVideo(id)
      toast.success('已通过')
      setVideos((prev) => prev.filter((v) => v.id !== id))
    } catch (err: any) {
      toast.error(err.response?.data?.error || '操作失败')
    }
  }

  const handleReject = async () => {
    if (!rejectModal) return
    try {
      await reviewAPI.rejectVideo(rejectModal.id, rejectModal.reason)
      toast.success('已拒绝')
      setVideos((prev) => prev.filter((v) => v.id !== rejectModal.id))
      setRejectModal(null)
    } catch (err: any) {
      toast.error(err.response?.data?.error || '操作失败')
    }
  }

  const saveConfig = async () => {
    if (!cfg) return
    setCfgSaving(true)
    try {
      await reviewAPI.updateConfig({
        enabled: cfg.enabled,
        text_model: cfg.text_model,
        vision_model: cfg.vision_model,
        sample_frames: cfg.sample_frames,
        frame_review_mode: cfg.frame_review_mode,
        confidence_threshold: cfg.confidence_threshold,
        manual_review_threshold: cfg.manual_review_threshold,
        max_retries: cfg.max_retries,
      })
      toast.success('配置已保存')
    } catch (err: any) {
      toast.error(err.response?.data?.error || '保存失败')
    } finally {
      setCfgSaving(false)
    }
  }

  const quickReasons = ['涉政', '色情', '暴力', '辱骂', '广告', '其他']

  return (
    <div className="max-w-5xl mx-auto">
      <h1 className="text-xl font-bold text-weibo-text mb-4">内容审核管理</h1>

      {/* Tabs */}
      <div className="flex gap-1 mb-4 bg-weibo-bg rounded-lg p-1 w-fit">
        <button
          onClick={() => setActiveTab('queue')}
          className={`px-4 py-1.5 rounded-md text-sm font-medium transition-colors ${
            activeTab === 'queue' ? 'bg-white text-weibo-text shadow-sm' : 'text-weibo-text-muted hover:text-weibo-text'
          }`}
        >
          待审核{videos.length > 0 ? ` (${videos.length})` : ''}
        </button>
        <button
          onClick={() => setActiveTab('config')}
          className={`px-4 py-1.5 rounded-md text-sm font-medium transition-colors ${
            activeTab === 'config' ? 'bg-white text-weibo-text shadow-sm' : 'text-weibo-text-muted hover:text-weibo-text'
          }`}
        >
          审核配置
        </button>
      </div>

      {/* Queue Tab */}
      {activeTab === 'queue' && (
        <>
          {loading ? (
            <div className="text-center py-12 text-weibo-text-muted">加载中...</div>
          ) : videos.length === 0 ? (
            <div className="text-center py-12 text-weibo-text-muted text-sm">暂无待审核内容</div>
          ) : (
            <div className="space-y-3">
              {videos.map((v) => (
                <div key={v.id} className="card flex gap-4">
                  {/* Cover thumbnail */}
                  <div className="relative w-32 h-20 shrink-0 rounded-lg overflow-hidden bg-weibo-bg">
                    <img
                      src={v.cover_url}
                      alt={v.title}
                      className="w-full h-full object-cover"
                    />
                    <button
                      onClick={() => setViewVideoId(viewVideoId === v.id ? null : v.id)}
                      className="absolute inset-0 bg-black/0 hover:bg-black/30 transition-colors flex items-center justify-center group"
                    >
                      <svg className="w-6 h-6 text-white opacity-0 group-hover:opacity-100 transition-opacity" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    </button>
                  </div>

                  {/* Info */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <p className="text-sm font-semibold text-weibo-text truncate">{v.title}</p>
                        {v.description && (
                          <p className="text-xs text-weibo-text-muted mt-0.5 line-clamp-1">{v.description}</p>
                        )}
                        <p className="text-2xs text-weibo-text-muted mt-1">
                          {v.username} · {formatDate(v.create_time)}
                        </p>
                      </div>
                      <div className="flex items-center gap-1.5 shrink-0">
                        <button
                          onClick={() => handleApprove(v.id)}
                          className="text-xs bg-green-500 text-white px-3 py-1 rounded-full font-medium hover:bg-green-600 transition-colors"
                        >
                          通过
                        </button>
                        <button
                          onClick={() => setRejectModal({ id: v.id, reason: '' })}
                          className="text-xs bg-red-500 text-white px-3 py-1 rounded-full font-medium hover:bg-red-600 transition-colors"
                        >
                          拒绝
                        </button>
                      </div>
                    </div>

                    {/* AI review info */}
                    {v.review_confidence !== undefined && v.review_confidence > 0 && (
                      <div className="flex items-center gap-2 mt-1.5 text-2xs">
                        <span className="text-weibo-text-muted">
                          AI置信度:
                          <span className={`ml-0.5 font-medium ${
                            v.review_confidence >= 0.7 ? 'text-green-500' :
                            v.review_confidence >= 0.5 ? 'text-yellow-500' : 'text-red-500'
                          }`}>
                            {(v.review_confidence * 100).toFixed(0)}%
                          </span>
                        </span>
                        {v.review_categories && (
                          <span className="text-weibo-text-muted">
                            检测: <span className="text-weibo-primary">{v.review_categories}</span>
                          </span>
                        )}
                      </div>
                    )}

                    {/* Video player */}
                    {viewVideoId === v.id && (
                      <div className="mt-2">
                        <video
                          src={v.play_url}
                          controls
                          className="w-full max-h-64 rounded-lg bg-black"
                        />
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      )}

      {/* Config Tab */}
      {activeTab === 'config' && cfg && (
        <div className="card space-y-4 max-w-lg">
          {/* Enable toggle */}
          <div className="flex items-center justify-between">
            <label className="text-sm font-medium text-weibo-text">启用内容审核</label>
            <button
              onClick={() => setCfg({ ...cfg, enabled: !cfg.enabled })}
              className={`w-10 h-5 rounded-full transition-colors relative ${cfg.enabled ? 'bg-green-500' : 'bg-gray-300'}`}
            >
              <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${cfg.enabled ? 'translate-x-5' : 'translate-x-0.5'}`} />
            </button>
          </div>

          {/* Text model */}
          <div>
            <label className="block text-xs font-medium text-weibo-text-secondary mb-1">文本审核模型</label>
            <input
              type="text"
              value={cfg.text_model}
              onChange={(e) => setCfg({ ...cfg, text_model: e.target.value })}
              className="input-field text-sm"
              disabled={!cfg.enabled}
            />
          </div>

          {/* Vision model */}
          <div>
            <label className="block text-xs font-medium text-weibo-text-secondary mb-1">视觉审核模型</label>
            <input
              type="text"
              value={cfg.vision_model}
              onChange={(e) => setCfg({ ...cfg, vision_model: e.target.value })}
              className="input-field text-sm"
              disabled={!cfg.enabled}
            />
          </div>

          {/* Thresholds */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium text-weibo-text-secondary mb-1">置信度阈值</label>
              <input
                type="number"
                value={cfg.confidence_threshold}
                onChange={(e) => setCfg({ ...cfg, confidence_threshold: Number(e.target.value) })}
                min={0.5} max={1.0} step={0.05}
                className="input-field text-sm"
                disabled={!cfg.enabled}
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-weibo-text-secondary mb-1">灰区下限</label>
              <input
                type="number"
                value={cfg.manual_review_threshold}
                onChange={(e) => setCfg({ ...cfg, manual_review_threshold: Number(e.target.value) })}
                min={0.3} max={0.8} step={0.05}
                className="input-field text-sm"
                disabled={!cfg.enabled}
              />
            </div>
          </div>

          {/* Max retries */}
          <div>
            <label className="block text-xs font-medium text-weibo-text-secondary mb-1">AI 最大重试次数</label>
            <input
              type="number"
              value={cfg.max_retries}
              onChange={(e) => setCfg({ ...cfg, max_retries: Number(e.target.value) })}
              min={0} max={5}
              className="input-field text-sm"
              disabled={!cfg.enabled}
            />
          </div>

          {/* Frame review mode */}
          <div>
            <label className="block text-xs font-medium text-weibo-text-secondary mb-1">帧审核模式</label>
            <select
              value={cfg.frame_review_mode}
              onChange={(e) => setCfg({ ...cfg, frame_review_mode: e.target.value })}
              className="input-field text-sm"
              disabled={!cfg.enabled}
            >
              <option value="off">关闭 (不抽帧)</option>
              <option value="on">开启 (始终抽帧)</option>
              <option value="auto">自动 (灰区时触发)</option>
            </select>
          </div>

          {/* Sample frames */}
          <div>
            <label className="block text-xs font-medium text-weibo-text-secondary mb-1">抽帧数量</label>
            <input
              type="number"
              value={cfg.sample_frames}
              onChange={(e) => setCfg({ ...cfg, sample_frames: Number(e.target.value) })}
              min={1} max={10}
              className="input-field text-sm"
              disabled={!cfg.enabled || cfg.frame_review_mode === 'off'}
            />
          </div>

          <button
            onClick={saveConfig}
            disabled={cfgSaving}
            className="btn-primary text-sm w-full"
          >
            {cfgSaving ? '保存中...' : '保存配置'}
          </button>
        </div>
      )}

      {/* Reject Modal */}
      {rejectModal && (
        <div className="fixed inset-0 bg-black/40 z-50 flex items-center justify-center" onClick={() => setRejectModal(null)}>
          <div className="bg-white rounded-xl p-5 w-full max-w-sm mx-4 shadow-xl" onClick={(e) => e.stopPropagation()}>
            <h3 className="font-semibold text-weibo-text mb-3">拒绝原因</h3>

            <div className="flex flex-wrap gap-1.5 mb-3">
              {quickReasons.map((r) => (
                <button
                  key={r}
                  onClick={() => setRejectModal((prev) => prev ? { ...prev, reason: prev.reason ? prev.reason + ', ' + r : r } : null)}
                  className={`text-xs px-2 py-1 rounded-full border transition-colors ${
                    rejectModal.reason.includes(r)
                      ? 'bg-weibo-primary text-white border-weibo-primary'
                      : 'border-weibo-border text-weibo-text-secondary hover:border-weibo-primary'
                  }`}
                >
                  {r}
                </button>
              ))}
            </div>

            <textarea
              value={rejectModal.reason}
              onChange={(e) => setRejectModal({ ...rejectModal, reason: e.target.value })}
              placeholder="补充说明..."
              className="input-field resize-none text-sm mb-3"
              rows={2}
            />

            <div className="flex gap-2 justify-end">
              <button onClick={() => setRejectModal(null)} className="btn-outline text-sm">取消</button>
              <button onClick={handleReject} className="bg-red-500 text-white px-4 py-1.5 rounded-full text-sm font-medium hover:bg-red-600 transition-colors">
                确认拒绝
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
