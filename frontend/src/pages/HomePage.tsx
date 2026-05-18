import { useEffect, useCallback } from 'react'
import { feedAPI } from '../api'
import { useFeedStore } from '../store/feedStore'
import { useInfiniteScroll } from '../hooks/useInfiniteScroll'
import FeedVideoCard from '../components/FeedVideoCard'

export default function HomePage() {
  const {
    videos, mode, loading, hasMore, nextTime, asOf, nextOffset,
    setMode, appendVideos, setLoading, setHasMore, setNextTime, setPagination, reset
  } = useFeedStore()

  const loadMore = useCallback(async () => {
    if (loading || !hasMore) return
    setLoading(true)
    try {
      if (mode === 'latest') {
        const { data } = await feedAPI.listLatest(10, nextTime)
        appendVideos(data.video_list)
        setHasMore(data.has_more)
        setNextTime(data.next_time)
      } else {
        const { data } = await feedAPI.listByPopularity(10, asOf, nextOffset)
        appendVideos(data.video_list)
        setHasMore(data.has_more)
        setPagination(data.as_of, data.next_offset)
      }
    } catch {
      // silent
    } finally {
      setLoading(false)
    }
  }, [mode, loading, hasMore, nextTime, asOf, nextOffset])

  useEffect(() => {
    reset()
  }, [mode])

  useEffect(() => {
    if (videos.length === 0 && hasMore) {
      loadMore()
    }
  }, [videos.length])

  const sentinelRef = useInfiniteScroll(loadMore, { enabled: hasMore && !loading })

  return (
    <div className="max-w-2xl mx-auto">
      {/* 模式切换 */}
      <div className="flex items-center gap-1 mb-4 bg-white rounded-full p-1 border border-weibo-border w-fit">
        {(['latest', 'popular'] as const).map((m) => (
          <button
            key={m}
            onClick={() => setMode(m)}
            className={`px-6 py-1.5 rounded-full text-sm font-medium transition-all duration-200 ${
              mode === m
                ? 'bg-weibo-primary text-white shadow-sm'
                : 'text-weibo-text-secondary hover:text-weibo-text'
            }`}
          >
            {m === 'latest' ? '最新' : '热门'}
          </button>
        ))}
      </div>

      {/* Feed流 */}
      <div className="space-y-3">
        {videos.map((v) => (
          <FeedVideoCard key={v.id} video={v} />
        ))}

        {/* 底部哨兵 */}
        <div ref={sentinelRef} className="py-4 text-center">
          {loading && (
            <div className="flex items-center justify-center gap-2 text-weibo-text-muted text-sm">
              <svg className="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              加载中...
            </div>
          )}
          {!hasMore && videos.length > 0 && (
            <p className="text-weibo-text-muted text-sm">— 已经到底了 —</p>
          )}
          {!loading && !hasMore && videos.length === 0 && (
            <div className="py-16 text-center text-weibo-text-muted">
              <p className="text-lg mb-2">📭</p>
              <p>暂无内容，快去发布第一个视频吧!</p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
