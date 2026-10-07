import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react'
import {
  UNAUTH_EVENT,
  authService,
  clearToken,
  getToken,
  setToken,
} from '../services/api.js'

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  const [token, setTokenState] = useState(() => getToken())
  const [loading, setLoading] = useState(true)

  const logout = useCallback(() => {
    clearToken()
    setTokenState(null)
    setUser(null)
  }, [])

  // Validasi token tersimpan saat aplikasi dimuat.
  useEffect(() => {
    let cancelled = false
    async function bootstrap() {
      const saved = getToken()
      if (!saved) {
        setLoading(false)
        return
      }
      try {
        const { data } = await authService.me()
        if (!cancelled) {
          setUser(data)
          setTokenState(saved)
        }
      } catch {
        if (!cancelled) logout()
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    bootstrap()
    return () => {
      cancelled = true
    }
  }, [logout])

  // Reset state saat interceptor mendeteksi 401 (token expired/invalid).
  useEffect(() => {
    window.addEventListener(UNAUTH_EVENT, logout)
    return () => window.removeEventListener(UNAUTH_EVENT, logout)
  }, [logout])

  const login = useCallback(async (email, password) => {
    const { data } = await authService.login({ email, password })
    setToken(data.token)
    setTokenState(data.token)
    try {
      const me = await authService.me()
      setUser(me.data)
    } catch {
      setUser(data.user)
    }
    return data.user
  }, [])

  const register = useCallback(
    async (name, email, password) => {
      const { data } = await authService.register({ name, email, password })
      setToken(data.token)
      setTokenState(data.token)
      setUser(data.user)
      return data.user
    },
    [],
  )

  // Memuat ulang profil dari /auth/me (mis. setelah update profil/avatar).
  const refreshUser = useCallback(async () => {
    const { data } = await authService.me()
    setUser(data)
    return data
  }, [])

  const value = useMemo(
    () => ({
      user,
      token,
      isAuthenticated: Boolean(token && user),
      loading,
      login,
      register,
      logout,
      refreshUser,
    }),
    [user, token, loading, login, register, logout, refreshUser],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

// eslint-disable-next-line react/only-export-components -- hook pendamping provider
export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth harus dipakai di dalam <AuthProvider>')
  return ctx
}
