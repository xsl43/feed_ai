import { useState, useEffect, useRef } from 'react'

import { mediaAPI, aiAPI, reviewAPI } from '../api'
import type { AIConfig, AIConfigUpdate, ReviewConfig } from '../api'
import type { MediaFile } from '../types'
import { useAuthStore } from '../store/authStore'
import toast from 'react-hot-toast'
import { formatDate } from '../utils/time'

export default function AIPage() {
  const [files, setFiles] = useState<MediaFile[]>([])
  const [loading, setLoading] = useState(true)
  const [summarizeText, setSummarizeText] = useState('')
  const [summary, setSummary] = useState('')
  const [summarizing, setSummarizing] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [dragOver, setDragOver] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const { user, isLoggedIn } = useAuthStore()

  // 阻止浏览器默认拖拽行为
  useEffect(() => {
    const preventDefaults = (e: DragEvent) => { e.preventDefault(); e.stopPropagation() }
    window.addEventListener('dragover', preventDefaults)
    window.addEventListener('drop', preventDefaults)
    return () => {
      window.removeEventListener('dragover', preventDefaults)
      window.removeEventListener('drop', preventDefaults)
    }
  }, [])

  // ──────────── AI 配置面板状态 ────────────
  const [configOpen, setConfigOpen] = useState(false)
  const [configLoading, setConfigLoading] = useState(false)
  const [configSaving, setConfigSaving] = useState(false)
  const [apiKey, setApiKey] = useState('')
  const [baseUrl, setBaseUrl] = useState('https://api.siliconflow.cn/v1')
  const [model, setModel] = useState('deepseek-ai/DeepSeek-V3')
  const [asrModel, setAsrModel] = useState('FunAudioLLM/SenseVoiceSmall')
  const [savedConfig, setSavedConfig] = useState<AIConfig | null>(null)
  const [showKey, setShowKey] = useState(false)

  // ──────────── 审核配置状态 ────────────
  const [reviewCfg, setReviewCfg] = useState<ReviewConfig | null>(null)

  // 加载已保存的配置
  const loadConfig = async () => {
    setConfigLoading(true)
    try {
      const { data } = await aiAPI.getConfig()
      setSavedConfig(data)
      if (data.base_url) setBaseUrl(data.base_url)
      if (data.model) setModel(data.model)
      if (data.asr_model) setAsrModel(data.asr_model)
    } catch {
      // 未配置过，使用默认值
    } finally {
      setConfigLoading(false)
    }
  }

  // 保存配置
  const saveConfig = async () => {
    setConfigSaving(true)
    try {
      const payload: AIConfigUpdate = { base_url: baseUrl, model, asr_model: asrModel }
      if (apiKey.trim()) payload.api_key = apiKey.trim()
      await aiAPI.updateConfig(payload)
      toast.success('AI 配置已保存')
      setApiKey('')
      await loadConfig()
    } catch (err: any) {
      toast.error(err.response?.data?.error || '保存失败')
    } finally {
      setConfigSaving(false)
    }
  }

  // 测试连接
  const testConnection = async () => {
    try {
      const payload: AIConfigUpdate = { base_url: baseUrl, model, asr_model: asrModel }
      if (apiKey.trim()) payload.api_key = apiKey.trim()
      await aiAPI.updateConfig(payload)
      await aiAPI.summarize('Hello, this is a test.')
      toast.success('连接测试通过！')
      setApiKey('')
      await loadConfig()
    } catch (err: any) {
      toast.error(err.response?.data?.error || '连接失败，请检查配置')
    }
  }

  // 加载审核配置
  const loadReviewConfig = async () => {
    try {
      const { data } = await reviewAPI.getConfig()
      setReviewCfg(data)
    } catch { /* use defaults */ }
  }

  // 保存审核配置


  useEffect(() => {
    if (!isLoggedIn) { setLoading(false); return }
    mediaAPI.list().then(({ data }) => {
      setFiles(data || [])
    }).catch(() => {}).finally(() => setLoading(false))
    loadConfig()
    loadReviewConfig()
  }, [isLoggedIn])

  const handleAnalyze = async (mediaId: number) => {
    try {
      await aiAPI.analyze(mediaId)
      toast.success('分析任务已提交，正在后台处理...')
      setTimeout(() => refreshStatus(mediaId), 3000)
    } catch (err: any) {
      toast.error(err.response?.data?.error || '提交失败')
    }
  }

  const handleDelete = async (mediaId: number) => {
    if (!confirm('确定要删除这条记录吗？')) return
    try {
      await mediaAPI.delete(mediaId, user?.id || 0)
      toast.success('已删除')
      setFiles((prev) => prev.filter((f) => f.id !== mediaId))
    } catch (err: any) {
      toast.error(err.response?.data?.error || '删除失败')
    }
  }

  const refreshStatus = async (mediaId: number) => {
    try {
      const { data } = await aiAPI.getStatus(mediaId)
      setFiles((prev) =>
        prev.map((f) =>
          f.id === mediaId ? { ...f, status: data.status, ai_summary: data.ai_summary, transcript_text: data.transcript_text } : f
        )
      )
      if (data.status === 'PROCESSING') {
        setTimeout(() => refreshStatus(mediaId), 5000)
      }
    } catch {}
  }

  const handleFileUpload = async (file: File) => {
    if (file.size > 500 * 1024 * 1024) {
      toast.error('文件大小不能超过 500MB')
      return
    }
    setUploading(true)
    try {
      await mediaAPI.upload(file, user?.id)
      toast.success('上传成功')
      const { data } = await mediaAPI.list(user?.id)
      setFiles(data || [])
    } catch (err: any) {
      toast.error(err.response?.data?.error || '上传失败')
    } finally {
      setUploading(false)
    }
  }

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0]
    if (!f) return
    handleFileUpload(f)
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDragOver(false)
    const f = e.dataTransfer.files?.[0]
    if (!f) return
    handleFileUpload(f)
  }

  const handleSummarize = async () => {
    if (!summarizeText.trim()) return
    setSummarizing(true)
    try {
      const { data } = await aiAPI.summarize(summarizeText.trim())
      setSummary(data.summary)
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'AI 总结失败')
    } finally {
      setSummarizing(false)
    }
  }

  if (!isLoggedIn) {
    return (
      <div className="text-center py-16 text-weibo-text-muted">
        请先登录后使用 AI 分析功能
      </div>
    )
  }

  return (
    <div className="max-w-4xl mx-auto">
      <h1 className="text-xl font-bold text-weibo-text mb-4">AI 智能分析</h1>

      {/* ──────────── API 配置面板 ──────────── */}
      <div className="card mb-4">
        <button
          onClick={() => { setConfigOpen(!configOpen); if (!configOpen && !savedConfig) loadConfig() }}
          className="w-full flex items-center justify-between"
        >
          <div className="flex items-center gap-2">
            <svg className="w-5 h-5 text-weibo-link" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            <span className="font-semibold text-weibo-text">API 接口配置</span>
            {savedConfig && (
              <span className="text-xs text-green-500 bg-green-50 px-2 py-0.5 rounded-full">
                已配置 {savedConfig.model}
              </span>
            )}
            {!savedConfig && !configLoading && (
              <span className="text-xs text-weibo-primary bg-weibo-highlight px-2 py-0.5 rounded-full">未配置</span>
            )}
          </div>
          <svg className={`w-4 h-4 text-weibo-text-muted transition-transform ${configOpen ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        {configOpen && (
          <div className="mt-4 pt-4 border-t border-weibo-border space-y-3">
            {/* Base URL */}
            <div>
              <label className="block text-xs font-medium text-weibo-text-secondary mb-1">
                Base URL <span className="text-weibo-text-muted font-normal">(API 端点地址)</span>
              </label>
              <input
                type="text"
                value={baseUrl}
                onChange={(e) => setBaseUrl(e.target.value)}
                placeholder="https://api.siliconflow.cn/v1"
                className="input-field text-sm"
              />
            </div>

            {/* Model */}
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-weibo-text-secondary mb-1">
                  Chat Model <span className="text-weibo-text-muted font-normal">(对话模型)</span>
                </label>
                <input
                  type="text"
                  value={model}
                  onChange={(e) => setModel(e.target.value)}
                  placeholder="deepseek-ai/DeepSeek-V3"
                  className="input-field text-sm"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-weibo-text-secondary mb-1">
                  ASR Model <span className="text-weibo-text-muted font-normal">(语音识别)</span>
                </label>
                <input
                  type="text"
                  value={asrModel}
                  onChange={(e) => setAsrModel(e.target.value)}
                  placeholder="FunAudioLLM/SenseVoiceSmall"
                  className="input-field text-sm"
                />
              </div>
            </div>

            {/* API Key */}
            <div>
              <label className="block text-xs font-medium text-weibo-text-secondary mb-1">
                API Key {savedConfig && <span className="text-weibo-text-muted font-normal">(留空则不修改)</span>}
              </label>
              <div className="relative">
                <input
                  type={showKey ? 'text' : 'password'}
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                  placeholder={savedConfig ? '•••••••• (留空保持不变)' : 'sk-xxxxxxxxxxxxxxxx'}
                  className="input-field text-sm pr-10"
                />
                <button
                  type="button"
                  onClick={() => setShowKey(!showKey)}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-weibo-text-muted hover:text-weibo-text"
                >
                  {showKey ? (
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                    </svg>
                  ) : (
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                    </svg>
                  )}
                </button>
              </div>
            </div>

            {/* 操作按钮 */}
            <div className="flex gap-2 pt-1">
              <button
                onClick={saveConfig}
                disabled={configSaving}
                className="btn-primary text-sm flex-1"
              >
                {configSaving ? '保存中...' : '保存配置'}
              </button>
              <button
                onClick={testConnection}
                disabled={configSaving}
                className="btn-outline text-sm"
              >
                测试连接
              </button>
            </div>

            {/* 当前配置提示 */}
            {savedConfig && (
              <div className="text-xs text-weibo-text-muted bg-weibo-bg rounded p-2 space-y-0.5">
                <p>当前：<span className="text-weibo-text-secondary">{savedConfig.base_url}</span></p>
                <p>模型：<span className="text-weibo-text-secondary">{savedConfig.model}</span> | ASR：<span className="text-weibo-text-secondary">{savedConfig.asr_model}</span></p>
              </div>
            )}
          </div>
        )}
      </div>

      {/* ──────────── 审核配置面板 ──────────── */}
      <div className="card mb-4">
        <button
          onClick={() => { if (!reviewCfg) loadReviewConfig() }}
          className="w-full flex items-center justify-between"
        >
          <div className="flex items-center gap-2">
            <svg className="w-5 h-5 text-weibo-link" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
            </svg>
            <span className="font-semibold text-weibo-text">内容审核状态</span>
            {reviewCfg && (
              <span className={`text-xs px-2 py-0.5 rounded-full ${reviewCfg.enabled ? 'text-green-500 bg-green-50' : 'text-yellow-500 bg-yellow-50'}`}>
                {reviewCfg.enabled ? '已启用' : '已禁用'}
              </span>
            )}
          </div>
        </button>

        {reviewCfg && (
          <div className="mt-4 pt-4 border-t border-weibo-border space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-weibo-text-muted">文本审核模型</span>
              <span className="text-weibo-text-secondary">{reviewCfg.text_model}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-weibo-text-muted">视觉审核模型</span>
              <span className="text-weibo-text-secondary">{reviewCfg.vision_model}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-weibo-text-muted">置信度阈值 / 灰区下限</span>
              <span className="text-weibo-text-secondary">{reviewCfg.confidence_threshold} / {reviewCfg.manual_review_threshold}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-weibo-text-muted">帧审核模式</span>
              <span className="text-weibo-text-secondary">
                {reviewCfg.frame_review_mode === 'off' ? '关闭' : reviewCfg.frame_review_mode === 'on' ? '开启' : '自动'}
              </span>
            </div>
          </div>
        )}
      </div>

      {/* 上传区域 */}
      <div className="card mb-4">
        <h2 className="font-semibold text-weibo-text mb-3">上传视频</h2>
        <input
          ref={fileInputRef}
          type="file"
          accept="video/mp4,video/mov,video/avi,video/mkv,video/webm"
          onChange={handleFileChange}
          className="hidden"
        />
        <div
          onClick={() => fileInputRef.current?.click()}
          onDragOver={(e) => { e.preventDefault(); e.stopPropagation(); setDragOver(true) }}
          onDragEnter={(e) => { e.preventDefault(); e.stopPropagation(); setDragOver(true) }}
          onDragLeave={(e) => { e.preventDefault(); e.stopPropagation(); setDragOver(false) }}
          onDrop={handleDrop}
          className={`border-2 border-dashed rounded-lg p-8 flex flex-col items-center justify-center cursor-pointer transition-colors ${
            dragOver ? 'border-weibo-primary bg-weibo-primary/5' : 'border-weibo-border-strong hover:border-weibo-primary'
          }`}
        >
          {uploading ? (
            <div className="text-center">
              <div className="animate-spin w-8 h-8 border-2 border-weibo-primary border-t-transparent rounded-full mx-auto mb-2" />
              <span className="text-sm text-weibo-text-muted">上传中...</span>
            </div>
          ) : (
            <div className="text-center text-weibo-text-muted">
              <svg className="w-10 h-10 mx-auto mb-2" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                <path d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
              </svg>
              <span className="text-sm">点击或拖拽上传视频</span>
              <span className="block text-xs text-weibo-text-muted mt-1">支持 MP4、MOV、AVI、MKV、WebM，最大 500MB</span>
            </div>
          )}
        </div>
      </div>

      {/* 媒体文件列表 */}
      <div className="card mb-4">
        <h2 className="font-semibold text-weibo-text mb-3">我的媒体文件</h2>
        {loading ? (
          <div className="text-center py-4 text-weibo-text-muted text-sm">加载中...</div>
        ) : files.length === 0 ? (
          <div className="text-center py-4 text-weibo-text-muted text-sm">暂无文件，请先上传视频</div>
        ) : (
          <div className="space-y-2">
            {files.map((f) => (
              <div key={f.id} className="flex items-center justify-between p-3 rounded-lg border border-weibo-border hover:border-weibo-primary transition-colors">
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-weibo-text truncate">{f.filename}</p>
                  <p className="text-2xs text-weibo-text-muted mt-0.5">
                    {formatDate(f.upload_time)} · {(f.file_size / 1024 / 1024).toFixed(1)}MB · 
                    <span className={`ml-1 font-medium ${
                      f.status === 'COMPLETED' ? 'text-green-500' :
                      f.status === 'PROCESSING' ? 'text-weibo-link' :
                      f.status === 'FAILED' ? 'text-weibo-primary' : 'text-weibo-text-muted'
                    }`}>
                      {f.status === 'UPLOADED' ? '已上传' :
                       f.status === 'PROCESSING' ? '分析中...' :
                       f.status === 'COMPLETED' ? '分析完成' :
                       f.status === 'FAILED' ? '分析失败' : f.status}
                    </span>
                  </p>
                  {f.ai_summary && (
                    <p className="text-xs text-weibo-text-secondary mt-1 line-clamp-2 bg-weibo-bg p-2 rounded">
                      <span className="font-medium">AI摘要：</span>{f.ai_summary}
                    </p>
                  )}
                </div>
                <button
                  onClick={() => handleAnalyze(f.id)}
                  disabled={f.status === 'PROCESSING'}
                  className="ml-3 btn-outline text-xs !px-3 !py-1 shrink-0"
                >
                  {f.status === 'COMPLETED' ? '重新分析' : f.status === 'PROCESSING' ? '处理中' : 'AI分析'}
                </button>
                <button
                  onClick={() => handleDelete(f.id)}
                  className="ml-1 text-red-400 hover:text-red-600 text-xs px-2 py-1 shrink-0"
                  title="删除"
                >
                  🗑
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* AI 文本总结 */}
      <div className="card">
        <h2 className="font-semibold text-weibo-text mb-3">AI 文本总结</h2>
        <textarea
          value={summarizeText}
          onChange={(e) => setSummarizeText(e.target.value)}
          className="input-field resize-none mb-3"
          rows={4}
          placeholder="输入任意文本，AI 将为你生成摘要..."
        />
        <button
          onClick={handleSummarize}
          disabled={!summarizeText.trim() || summarizing}
          className="btn-primary text-sm mb-3"
        >
          {summarizing ? 'AI思考中...' : '生成摘要'}
        </button>
        {summary && (
          <div className="bg-weibo-bg rounded-lg p-4">
            <p className="text-xs font-medium text-weibo-text-muted mb-1">AI 摘要</p>
            <p className="text-sm text-weibo-text leading-relaxed">{summary}</p>
          </div>
        )}
      </div>
    </div>
  )
}
