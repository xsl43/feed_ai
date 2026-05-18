import { useEffect, useRef, useCallback } from 'react'

export function useInfiniteScroll(
  callback: () => void,
  options: { enabled?: boolean; threshold?: number } = {}
) {
  const { enabled = true, threshold = 200 } = options
  const observerRef = useRef<HTMLDivElement | null>(null)

  const handleIntersect = useCallback(
    (entries: IntersectionObserverEntry[]) => {
      if (entries[0].isIntersecting && enabled) {
        callback()
      }
    },
    [callback, enabled]
  )

  useEffect(() => {
    const el = observerRef.current
    if (!el) return

    const observer = new IntersectionObserver(handleIntersect, {
      rootMargin: `${threshold}px`,
    })
    observer.observe(el)
    return () => observer.disconnect()
  }, [handleIntersect, threshold])

  return observerRef
}
