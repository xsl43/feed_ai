import { Link } from 'react-router-dom'
import type { FeedVideoItem } from '../types'
import { timeAgo, formatCount } from '../utils/time'

interface Props {
  video: FeedVideoItem
}

export default function FeedVideoCard({ video }: Props) {
  return (
    <article className="card hover:shadow-md transition-shadow duration-200">
      <div className="flex gap-3">
        {/* 头像 */}
        <Link to={`/profile/${video.author.id}`} className="shrink-0">
          <div className="w-10 h-10 rounded-full bg-gradient-to-br from-weibo-primary to-weibo-link text-white flex items-center justify-center text-sm font-bold">
            {video.author.username[0]?.toUpperCase() || 'U'}
          </div>
        </Link>

        {/* 内容区 */}
        <div className="flex-1 min-w-0">
          {/* 作者信息 */}
          <div className="flex items-center gap-2 mb-1.5">
            <Link
              to={`/profile/${video.author.id}`}
              className="font-medium text-weibo-text hover:text-weibo-primary transition-colors text-sm"
            >
              {video.author.username}
            </Link>
            <span className="text-weibo-text-muted text-2xs">
              {timeAgo(video.create_time)}
            </span>
          </div>

          {/* 标题 */}
          <Link to={`/video/${video.id}`}>
            <p className="text-weibo-text text-sm leading-relaxed mb-2 line-clamp-2 hover:text-weibo-primary transition-colors">
              {video.title}
            </p>
          </Link>

          {/* 视频封面 */}
          <Link to={`/video/${video.id}`} className="block relative rounded-lg overflow-hidden bg-weibo-bg mb-3 group">
            <img
              src={video.cover_url}
              alt={video.title}
              className="w-full aspect-video object-cover group-hover:scale-105 transition-transform duration-300"
              loading="lazy"
            />
            <div className="absolute inset-0 bg-black/0 group-hover:bg-black/10 transition-colors flex items-center justify-center">
              <div className="w-12 h-12 rounded-full bg-white/80 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
                <svg className="w-5 h-5 text-weibo-primary ml-0.5" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M8 5v14l11-7z" />
                </svg>
              </div>
            </div>
          </Link>

          {/* 互动栏 */}
          <div className="flex items-center gap-5 text-weibo-text-secondary text-xs">
            <button className="flex items-center gap-1 hover:text-weibo-primary transition-colors">
              <svg className="w-4 h-4" fill={video.is_liked ? '#E6162D' : 'none'} stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z" />
              </svg>
              {video.likes_count > 0 && <span>{formatCount(video.likes_count)}</span>}
            </button>
            <Link
              to={`/video/${video.id}`}
              className="flex items-center gap-1 hover:text-weibo-primary transition-colors"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
              </svg>
              评论
            </Link>
            <button className="flex items-center gap-1 hover:text-weibo-primary transition-colors">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                <path d="M17.593 3.322c1.1.128 1.907 1.077 1.907 2.185V21L12 17.25 4.5 21V5.507c0-1.108.806-2.057 1.907-2.185a48.507 48.507 0 0 1 11.186 0z" />
              </svg>
              收藏
            </button>
            <button className="flex items-center gap-1 hover:text-weibo-primary transition-colors">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                <circle cx="18" cy="5" r="3" />
                <circle cx="6" cy="12" r="3" />
                <circle cx="18" cy="19" r="3" />
                <line x1="8.59" y1="13.51" x2="15.42" y2="17.49" />
                <line x1="15.41" y1="6.51" x2="8.59" y2="10.49" />
              </svg>
              分享
            </button>
          </div>
        </div>
      </div>
    </article>
  )
}
