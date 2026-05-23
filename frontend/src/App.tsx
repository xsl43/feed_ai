import { Routes, Route } from 'react-router-dom'
import AppShell from './components/AppShell'
import HomePage from './pages/HomePage'
import LoginPage from './pages/LoginPage'
import VideoDetailPage from './pages/VideoDetailPage'
import ProfilePage from './pages/ProfilePage'
import PublishPage from './pages/PublishPage'
import MessagesPage from './pages/MessagesPage'
import AdminReviewPage from './pages/AdminReviewPage'
import AIPage from './pages/AIPage'

export default function App() {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route path="/" element={<HomePage />} />
        <Route path="/video/:id" element={<VideoDetailPage />} />
        <Route path="/profile/:id" element={<ProfilePage />} />
        <Route path="/publish" element={<PublishPage />} />
        <Route path="/messages" element={<MessagesPage />} />
        <Route path="/ai" element={<AIPage />} />
        <Route path="/admin/review" element={<AdminReviewPage />} />
      </Route>
      <Route path="/login" element={<LoginPage />} />
    </Routes>
  )
}
