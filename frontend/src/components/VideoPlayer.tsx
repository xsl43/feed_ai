interface Props {
  src: string
  poster?: string
  className?: string
}

export default function VideoPlayer({ src, poster, className = '' }: Props) {
  return (
    <div className={`relative bg-black rounded-lg overflow-hidden ${className}`}>
      <video
        src={src}
        poster={poster}
        controls
        className="w-full max-h-[70vh] object-contain"
        playsInline
        preload="metadata"
      >
        您的浏览器不支持视频播放
      </video>
    </div>
  )
}
