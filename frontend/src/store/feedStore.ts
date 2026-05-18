import { create } from 'zustand'
import type { FeedVideoItem } from '../types'

type FeedMode = 'latest' | 'popular'

interface FeedState {
  videos: FeedVideoItem[]
  mode: FeedMode
  loading: boolean
  hasMore: boolean
  nextTime: number
  asOf: number
  nextOffset: number
  setMode: (mode: FeedMode) => void
  setVideos: (videos: FeedVideoItem[]) => void
  appendVideos: (videos: FeedVideoItem[]) => void
  setLoading: (loading: boolean) => void
  setHasMore: (hasMore: boolean) => void
  setNextTime: (t: number) => void
  setPagination: (asOf: number, nextOffset: number) => void
  reset: () => void
}

export const useFeedStore = create<FeedState>((set) => ({
  videos: [],
  mode: 'latest',
  loading: false,
  hasMore: true,
  nextTime: 0,
  asOf: 0,
  nextOffset: 0,

  setMode: (mode) => {
    set({ mode, videos: [], nextTime: 0, asOf: 0, nextOffset: 0, hasMore: true })
  },

  setVideos: (videos) => set({ videos }),
  appendVideos: (videos) => set((s) => {
    const existingIds = new Set(s.videos.map((v) => v.id))
    const newVideos = videos.filter((v) => !existingIds.has(v.id))
    return { videos: [...s.videos, ...newVideos] }
  }),
  setLoading: (loading) => set({ loading }),
  setHasMore: (hasMore) => set({ hasMore }),
  setNextTime: (nextTime) => set({ nextTime }),
  setPagination: (asOf, nextOffset) => set({ asOf, nextOffset }),

  reset: () =>
    set({ videos: [], nextTime: 0, asOf: 0, nextOffset: 0, hasMore: true, loading: false }),
}))
