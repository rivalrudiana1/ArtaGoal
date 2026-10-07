import axios from 'axios'

const AUTH_BASE_URL =
  import.meta.env.VITE_AUTH_API_URL ?? 'http://localhost:8081/api/v1'
const GOAL_BASE_URL =
  import.meta.env.VITE_GOAL_API_URL ?? 'http://localhost:8080/api/v1'

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

function createClient(baseURL) {
  const client = axios.create({ baseURL, timeout: 15000 })
  client.interceptors.request.use(attachAuthHeader)
  client.interceptors.response.use(
    (response) => response,
    handleUnauthorized,
  )
  return client
}

/** auth-service : register, login, me. */
export const authApi = createClient(AUTH_BASE_URL)
/** goal-service : goals, contributions, progress, projection. */
export const goalApi = createClient(GOAL_BASE_URL)

export const authService = {
  register: (payload) => authApi.post('/auth/register', payload),
  login: (payload) => authApi.post('/auth/login', payload),
  me: () => authApi.get('/auth/me'),
  // Header Content-Type multipart/form-data beserta boundary diatur
  // otomatis oleh axios saat body berupa FormData — jangan di-set manual
  // agar browser menyisipkan boundary yang benar.
  updateProfile: (formData) => authApi.put('/auth/profile', formData),
}

/** URL absolut file statis auth-service (mis. avatar) dari path relatifnya. */
export function authFileUrl(path) {
  if (!path) return ''
  if (/^https?:\/\//i.test(path)) return path
  const origin = AUTH_BASE_URL.replace(/\/api\/v1\/?$/, '')
  return `${origin}${path.startsWith('/') ? path : `/${path}`}`
}

export const goalService = {
  listMyGoals: () => goalApi.get('/goals'),
  getGoal: (id) => goalApi.get(`/goals/${id}`),
  createGoal: (payload) => goalApi.post('/goals', payload),
  updateGoal: (id, payload) => goalApi.put(`/goals/${id}`, payload),
  deleteGoal: (id) => goalApi.delete(`/goals/${id}`),
  addContribution: (goalId, payload) =>
    goalApi.post(`/goals/${goalId}/contributions`, payload),
  listContributions: (goalId) =>
    goalApi.get(`/goals/${goalId}/contributions`),
  deleteContribution: (goalId, contributionId) =>
    goalApi.delete(`/goals/${goalId}/contributions/${contributionId}`),
  getProgress: (goalId) => goalApi.get(`/goals/${goalId}/progress`),
  getProjection: (goalId) => goalApi.get(`/goals/${goalId}/projection`),
  getHeatmap: () => goalApi.get('/goals/heatmap'),
  getNotifications: () => goalApi.get('/notifications'),
  markAsRead: (id) => goalApi.put(`/notifications/${id}/read`),
}

/** Ambil pesan error backend {error: "..."} atau fallback generik. */
export function apiErrorMessage(error, fallback = 'Terjadi kesalahan.') {
  return error.response?.data?.error ?? error.message ?? fallback
}
