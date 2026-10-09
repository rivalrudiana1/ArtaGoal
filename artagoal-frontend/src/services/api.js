import axios from 'axios'

/** Hapus garis miring di akhir agar gabungan `${base}/path` selalu tepat satu `/`. */
function normalizeBaseUrl(value, fallback) {
  const raw = String(value || fallback || '').trim() || fallback
  return raw.replace(/\/+$/, '')
}

export const AUTH_BASE_URL = normalizeBaseUrl(
  import.meta.env.VITE_AUTH_API_URL,
  'http://localhost:8081/api/v1',
)
export const GOAL_BASE_URL = normalizeBaseUrl(
  import.meta.env.VITE_GOAL_API_URL,
  'http://localhost:8080/api/v1',
)

export const TOKEN_KEY = 'artagoal_token'
export const UNAUTH_EVENT = 'artagoal:unauthorized'

const PUBLIC_PATHS = ['/login', '/register']

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

function attachAuthHeader(config) {
  const token = getToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
}

/** 401 di luar halaman publik => bersihkan token & kembali ke /login. */
function handleUnauthorized(error) {
  if (error.response?.status === 401) {
    clearToken()
    window.dispatchEvent(new Event(UNAUTH_EVENT))
    if (!PUBLIC_PATHS.includes(window.location.pathname)) {
      window.location.href = '/login'
    }
  }
  return Promise.reject(error)
}

function createClient() {
  const client = axios.create({ timeout: 15000 })
  client.interceptors.request.use(attachAuthHeader)
  client.interceptors.response.use(
    (response) => response,
    handleUnauthorized,
  )
  return client
}

/** auth-service : register, login, me. URL penuh eksplisit, tanpa mengandalkan baseURL. */
export const authApi = createClient()
/** goal-service : goals, contributions, progress, projection. URL penuh eksplisit. */
export const goalApi = createClient()

export const authService = {
  register: (payload) =>
    authApi.post(`${AUTH_BASE_URL}/auth/register`, payload),
  login: (payload) => authApi.post(`${AUTH_BASE_URL}/auth/login`, payload),
  me: () => authApi.get(`${AUTH_BASE_URL}/auth/me`),
  // Header Content-Type multipart/form-data beserta boundary diatur
  // otomatis oleh axios saat body berupa FormData — jangan di-set manual
  // agar browser menyisipkan boundary yang benar.
  updateProfile: (formData) =>
    authApi.put(`${AUTH_BASE_URL}/auth/profile`, formData),
  changePassword: (payload) =>
    authApi.put(`${AUTH_BASE_URL}/auth/password`, payload),
}

/** URL absolut file statis auth-service (mis. avatar) dari path relatifnya. */
export function authFileUrl(path) {
  if (!path) return ''
  if (/^https?:\/\//i.test(path)) return path
  const origin = AUTH_BASE_URL.replace(/\/api\/v1\/?$/, '')
  return `${origin}${path.startsWith('/') ? path : `/${path}`}`
}

export const goalService = {
  listMyGoals: (params) => goalApi.get(`${GOAL_BASE_URL}/goals`, { params }),
  getGoal: (id) => goalApi.get(`${GOAL_BASE_URL}/goals/${id}`),
  createGoal: (payload) => goalApi.post(`${GOAL_BASE_URL}/goals`, payload),
  updateGoal: (id, payload) =>
    goalApi.put(`${GOAL_BASE_URL}/goals/${id}`, payload),
  deleteGoal: (id) => goalApi.delete(`${GOAL_BASE_URL}/goals/${id}`),
  addContribution: (goalId, payload) =>
    goalApi.post(`${GOAL_BASE_URL}/goals/${goalId}/contributions`, payload),
  listContributions: (goalId, params) =>
    goalApi.get(`${GOAL_BASE_URL}/goals/${goalId}/contributions`, { params }),
  deleteContribution: (goalId, contributionId) =>
    goalApi.delete(
      `${GOAL_BASE_URL}/goals/${goalId}/contributions/${contributionId}`,
    ),
  getProgress: (goalId) =>
    goalApi.get(`${GOAL_BASE_URL}/goals/${goalId}/progress`),
  getProjection: (goalId) =>
    goalApi.get(`${GOAL_BASE_URL}/goals/${goalId}/projection`),
  getHeatmap: () => goalApi.get(`${GOAL_BASE_URL}/goals/heatmap`),
  getStats: () => goalApi.get(`${GOAL_BASE_URL}/goals/stats`),
  getNotifications: (params) =>
    goalApi.get(`${GOAL_BASE_URL}/notifications`, { params }),
  markAsRead: (id) => goalApi.put(`${GOAL_BASE_URL}/notifications/${id}/read`),
  getVapidPublicKey: () =>
    goalApi.get(`${GOAL_BASE_URL}/push/vapid-public-key`),
  savePushSubscription: (payload) =>
    goalApi.post(`${GOAL_BASE_URL}/push/subscriptions`, payload),
  deletePushSubscription: (endpoint) =>
    goalApi.delete(`${GOAL_BASE_URL}/push/subscriptions`, {
      data: { endpoint },
    }),
}

/** Ambil pesan error backend {error: "..."} atau fallback generik. */
export function apiErrorMessage(error, fallback = 'Terjadi kesalahan.') {
  return error?.response?.data?.error ?? error?.message ?? fallback
}
