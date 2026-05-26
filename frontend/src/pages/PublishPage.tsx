import { useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { videoAPI, reviewAPI } from '../api'
import type { ReviewStatus } from '../api/review'
import { useAuthStore } from '../store/authStore'
import toast from 'react-hot-toast'

export default function PublishPage() {
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [videoFile, setVideoFile] = useState<File | null>(null)
  const [coverFile, setCoverFile] = useState<File | null>(null)
  const [videoPreview, setVideoPreview] = useState('')
  const [coverPreview, setCoverPreview] = useState('')
  const [uploading, setUploading] = useState(false)
  const [publishing, setPublishing] = useState(false)
  const [publishResult, setPublishResult] = useState<{ id: number; review_status: string } | null>(null)
  const [reviewStatus, setReviewStatus] = useState<ReviewStatus | null>(null)
  const { isLoggedIn } = useAuthStore()
  const navigate = useNavigate()
  const videoInputRef = useRef<HTMLInputElement>(null)
  const coverInputRef = useRef<HTMLInputElement>(null)

  const handleVideoChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0]
    if (!f) return
    if (f.size > 500 * 1024 * 1024) {
      toast.error('视频文件大小不能超过 500MB')
      return
    }
    setVideoFile(f)
    setVideoPreview(URL.createObjectURL(f))
  }

  const handleCoverChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0]
    if (!f) return
    if (!['image/jpeg', 'image/png', 'image/webp'].includes(f.type)) {
      toast.error('封面仅支持 JPG/PNG/WebP 格式')
      return
    }
    setCoverFile(f)
    setCoverPreview(URL.createObjectURL(f))
  }

  const pollReviewStatus = async (videoId: number) => {
    const poll = async () => {
      try {
        const { data } = await reviewAPI.getStatus(videoId)
        setReviewStatus(data)
        if (data.review_status === 'approved') {
          toast.success('审核通过! 视频已发布')
          setTimeout(() => navigate(`/video/${videoId}`), 1500)
          return
        }
        if (data.review_status === 'rejected') {
          toast.error(`审核未通过: ${data.review_reason || '违规内容'}`)
          return
        }
        setTimeout(() => poll(), 3000)
      } catch {
        setTimeout(() => poll(), 3000)
      }
    }
    poll()
  }

  const handlePublish = async () => {
    if (!isLoggedIn) { toast.error('请先登录'); return }
    if (!title.trim()) { toast.error('请输入标题'); return }
    if (!videoFile) { toast.error('请选择视频'); return }
    if (!coverFile) { toast.error('请选择封面'); return }

    setUploading(true)
    try {
      toast.loading('正在上传视频...')
      const vRes = await videoAPI.uploadVideo(videoFile)
      const playUrl = vRes.data.play_url
      toast.dismiss()

      toast.loading('正在上传封面...')
      const cRes = await videoAPI.uploadCover(coverFile)
      const coverUrl = cRes.data.cover_url
      toast.dismiss()

      setPublishing(true)
      const pubRes = await videoAPI.publish({
        title: title.trim(),
        description: description.trim(),
        play_url: playUrl,
        cover_url: coverUrl,
      })

      const videoData = pubRes.data
      setPublishResult({ id: videoData.id, review_status: videoData.review_status || 'pending' })

      if (!videoData.review_status || videoData.review_status === 'pending') {
        toast.success('视频已提交，正在审核中...')
        pollReviewStatus(videoData.id)
      } else {
        toast.success('发布成功!')
        navigate('/')
      }
    } catch (err: any) {
      toast.dismiss()
      toast.error(err.response?.data?.error || '发布失败')
    } finally {
      setUploading(false)
      setPublishing(false)
    }
  }

  return (
    <div className="max-w-2xl mx-auto">
      <h1 className="text-xl font-bold text-weibo-text mb-4">发布视频</h1>

      <div className="card space-y-4">
        {/* 标题 */}
        <div>
          <label className="block text-sm font-medium text-weibo-text mb-1">标题</label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="input-field"
            placeholder="给你的视频取个吸引人的标题..."
            maxLength={100}
          />
        </div>

        {/* 描述 */}
        <div>
          <label className="block text-sm font-medium text-weibo-text mb-1">描述</label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="input-field resize-none"
            rows={3}
            placeholder="简单描述一下视频内容..."
            maxLength={255}
          />
        </div>

        {/* 上传区域 */}
        <div className="grid grid-cols-2 gap-4">
          {/* 视频上传 */}
          <div>
            <label className="block text-sm font-medium text-weibo-text mb-2">视频文件</label>
            <input
              ref={videoInputRef}
              type="file"
              accept="video/mp4,video/mov,video/avi,video/mkv,video/webm"
              onChange={handleVideoChange}
              className="hidden"
            />
            <div
              onClick={() => videoInputRef.current?.click()}
              className={`border-2 border-dashed rounded-lg flex flex-col items-center justify-center cursor-pointer transition-colors aspect-video ${
                videoPreview ? 'border-weibo-primary' : 'border-weibo-border-strong hover:border-weibo-primary'
              }`}
            >
              {videoPreview ? (
                <video src={videoPreview} className="w-full h-full object-cover rounded-lg" />
              ) : (
                <div className="text-center text-weibo-text-muted">
                  <svg className="w-8 h-8 mx-auto mb-1" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                    <path d="M12 5v14M5 12h14" />
                  </svg>
                  <span className="text-xs">点击上传 MP4</span>
                </div>
              )}
            </div>
            {videoFile && <p className="text-2xs text-weibo-text-muted mt-1 truncate">{videoFile.name}</p>}
          </div>

          {/* 封面上传 */}
          <div>
            <label className="block text-sm font-medium text-weibo-text mb-2">封面图片</label>
            <input
              ref={coverInputRef}
              type="file"
              accept="image/jpeg,image/png,image/webp"
              onChange={handleCoverChange}
              className="hidden"
            />
            <div
              onClick={() => coverInputRef.current?.click()}
              className={`border-2 border-dashed rounded-lg flex flex-col items-center justify-center cursor-pointer transition-colors aspect-video ${
                coverPreview ? 'border-weibo-primary' : 'border-weibo-border-strong hover:border-weibo-primary'
              }`}
            >
              {coverPreview ? (
                <img src={coverPreview} alt="封面预览" className="w-full h-full object-cover rounded-lg" />
              ) : (
                <div className="text-center text-weibo-text-muted">
                  <svg className="w-8 h-8 mx-auto mb-1" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                    <path d="M12 5v14M5 12h14" />
                  </svg>
                  <span className="text-xs">点击上传封面</span>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* 审核状态 */}
        {publishResult && (
          <div className={`p-3 rounded-lg text-sm ${
            reviewStatus?.review_status === 'approved' ? 'bg-green-50 text-green-700' :
            reviewStatus?.review_status === 'rejected' ? 'bg-red-50 text-red-700' :
            'bg-blue-50 text-blue-700'
          }`}>
            {reviewStatus?.review_status === 'approved' && <p>审核已通过，即将跳转...</p>}
            {reviewStatus?.review_status === 'rejected' && (
              <div>
                <p className="font-medium">审核未通过</p>
                {reviewStatus.review_reason && <p className="text-xs mt-1">{reviewStatus.review_reason}</p>}
                <button
                  onClick={() => { setPublishResult(null); setReviewStatus(null) }}
                  className="mt-2 text-xs underline"
                >
                  修改后重新提交
                </button>
              </div>
            )}
            {(!reviewStatus || reviewStatus.review_status === 'pending') && (
              <p>正在审核中，请稍候...</p>
            )}
          </div>
        )}

        {/* 发布按钮 */}
        <button
          onClick={handlePublish}
          disabled={uploading || publishing || !title.trim() || !videoFile || !coverFile}
          className="btn-primary w-full !py-3 text-base"
        >
          {uploading ? '上传中...' : publishing ? '发布中...' : '发布视频'}
        </button>
      </div>
    </div>
  )
}
