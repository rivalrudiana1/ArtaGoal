import { Navigate, Route, Routes } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'
import DashboardPage from '../pages/DashboardPage.jsx'
import LoginPage from '../pages/LoginPage.jsx'
import NewGoalPage from '../pages/NewGoalPage.jsx'
import ProfilePage from '../pages/ProfilePage.jsx'
import RegisterPage from '../pages/RegisterPage.jsx'

function FullPageLoader() {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="h-10 w-10 animate-spin rounded-full border-2 border-slate-200 border-t-emerald-600" />
    </div>
  )
}

/** Menjaga halaman privat: lempar ke /login bila belum terautentikasi. */
function ProtectedRoute({ children }) {
  const { isAuthenticated, loading } = useAuth()
  if (loading) return <FullPageLoader />
  if (!isAuthenticated) return <Navigate to="/login" replace />
  return children
}

/** Menjaga halaman auth: lempar ke /dashboard bila sudah login. */
function GuestRoute({ children }) {
  const { isAuthenticated, loading } = useAuth()
  if (loading) return <FullPageLoader />
  if (isAuthenticated) return <Navigate to="/dashboard" replace />
  return children
}

function NotFound() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-2">
      <p className="text-4xl font-bold text-slate-800">404</p>
      <p className="text-slate-500">Halaman tidak ditemukan.</p>
    </div>
  )
}

export default function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/dashboard" replace />} />
      <Route
        path="/login"
        element={
          <GuestRoute>
            <LoginPage />
          </GuestRoute>
        }
      />
      <Route
        path="/register"
        element={
          <GuestRoute>
            <RegisterPage />
          </GuestRoute>
        }
      />
      <Route
        path="/dashboard"
        element={
          <ProtectedRoute>
            <DashboardPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/goals/new"
        element={
          <ProtectedRoute>
            <NewGoalPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/profile"
        element={
          <ProtectedRoute>
            <ProfilePage />
          </ProtectedRoute>
        }
      />
      <Route path="*" element={<NotFound />} />
    </Routes>
  )
}
