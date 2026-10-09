import axios from 'axios'

const ensureApiPrefix = (url, defaultUrl) => {
  const baseUrl = url || defaultUrl
  if (!baseUrl) return ''
  const clean = baseUrl.replace(/\/+$/, '')
  return clean.endsWith('/api/v1') ? clean : `${clean}/api/v1`
}

const joinUrl = (baseUrl, endpoint) => {
  if (!baseUrl) return endpoint
  const cleanBase = baseUrl.replace(/\/+$/, '')
  const cleanEndpoint = endpoint.replace(/^\/+/, '')
  return `${cleanBase}/${cleanEndpoint}`
}

const AUTH_BASE_URL = ensureApiPrefix(
  import.meta.env.VITE_AUTH_API_URL,
  'https://auth-service-production-bc66.up.railway.app/api/v1',
)
const GOAL_BASE_URL = ensureApiPrefix(
  import.meta.env.VITE_GOAL_API_URL,
  'https://goal-service-production.up.railway.app/api/v1',
)

console.log('[API BASE AUTH]:', AUTH_BASE_URL)

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

axios.defaults.timeout = 15000
axios.interceptors.request.use(attachAuthHeader)
axios.interceptors.response.use(
  (response) => response,
  handleUnauthorized,
)

export const authService = {
  register: (payload) =>
    axios.post(joinUrl(AUTH_BASE_URL, 'auth/register'), payload),
  login: (payload) => {
    console.log('[DEBUG API] Login URL:', joinUrl(AUTH_BASE_URL, 'auth/login'))
    return axios.post(joinUrl(AUTH_BASE_URL, 'auth/login'), payload)
  },
  me: () => axios.get(joinUrl(AUTH_BASE_URL, 'auth/me')),
  // Header Content-Type multipart/form-data beserta boundary diatur
  // otomatis oleh axios saat body berupa FormData — jangan di-set manual
  // agar browser menyisipkan boundary yang benar.
  updateProfile: (formData) =>
    axios.put(joinUrl(AUTH_BASE_URL, 'auth/profile'), formData),
  changePassword: (payload) =>
    axios.put(joinUrl(AUTH_BASE_URL, 'auth/password'), payload),
}

/** URL absolut file statis auth-service (mis. avatar) dari path relatifnya. */
export function authFileUrl(path) {
  if (!path) return ''
  if (/^https?:\/\//i.test(path)) return path
  const origin = AUTH_BASE_URL.replace(/\/api\/v1\/?$/, '')
  return `${origin}${path.startsWith('/') ? path : `/${path}`}`
}

export const goalService = {
  listMyGoals: (params) =>
    axios.get(joinUrl(GOAL_BASE_URL, 'goals'), { params }),
  getGoal: (id) => axios.get(joinUrl(GOAL_BASE_URL, `goals/${id}`)),
  createGoal: (payload) =>
    axios.post(joinUrl(GOAL_BASE_URL, 'goals'), payload),
  updateGoal: (id, payload) =>
    axios.put(joinUrl(GOAL_BASE_URL, `goals/${id}`), payload),
  deleteGoal: (id) => axios.delete(joinUrl(GOAL_BASE_URL, `goals/${id}`)),
  addContribution: (goalId, payload) =>
    axios.post(
      joinUrl(GOAL_BASE_URL, `goals/${goalId}/contributions`),
      payload,
    ),
  listContributions: (goalId, params) =>
    axios.get(joinUrl(GOAL_BASE_URL, `goals/${goalId}/contributions`), {
      params,
    }),
  deleteContribution: (goalId, contributionId) =>
    axios.delete(
      joinUrl(GOAL_BASE_URL, `goals/${goalId}/contributions/${contributionId}`),
    ),
  getProgress: (goalId) =>
    axios.get(joinUrl(GOAL_BASE_URL, `goals/${goalId}/progress`)),
  getProjection: (goalId) =>
    axios.get(joinUrl(GOAL_BASE_URL, `goals/${goalId}/projection`)),
  getHeatmap: () => axios.get(joinUrl(GOAL_BASE_URL, 'goals/heatmap')),
  getStats: () => axios.get(joinUrl(GOAL_BASE_URL, 'goals/stats')),
  getNotifications: (params) =>
    axios.get(joinUrl(GOAL_BASE_URL, 'notifications'), { params }),
  markAsRead: (id) =>
    axios.put(joinUrl(GOAL_BASE_URL, `notifications/${id}/read`)),
  getVapidPublicKey: () =>
    axios.get(joinUrl(GOAL_BASE_URL, 'push/vapid-public-key')),
  savePushSubscription: (payload) =>
    axios.post(joinUrl(GOAL_BASE_URL, 'push/subscriptions'), payload),
  deletePushSubscription: (endpoint) =>
    axios.delete(joinUrl(GOAL_BASE_URL, 'push/subscriptions'), {
      data: { endpoint },
    }),
}

/** Ambil pesan error backend {error: "..."} atau fallback generik. */
export function apiErrorMessage(error, fallback = 'Terjadi kesalahan.') {
  return error?.response?.data?.error ?? error?.message ?? fallback
}
