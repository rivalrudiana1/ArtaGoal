import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ArrowLeft, Camera, Loader2, LogOut, PiggyBank } from 'lucide-react'
import { toast } from 'sonner'
import { useAuth } from '../context/AuthContext.jsx'
import ThemeToggle from '../components/ThemeToggle.jsx'
import BottomNav from '../components/BottomNav.jsx'
import ConfirmModal from '../components/ui/ConfirmModal.jsx'
import { apiErrorMessage, authFileUrl, authService } from '../services/api.js'

const MAX_AVATAR_SIZE = 2 * 1024 * 1024

function initials(name) {
  const parts = String(name ?? '').trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

export default function ProfilePage() {
  const { user, logout, refreshUser } = useAuth()
  const navigate = useNavigate()
  const [name, setName] = useState(user?.name ?? '')
  const [file, setFile] = useState(null)
  const [preview, setPreview] = useState('')
  const [saving, setSaving] = useState(false)
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [changingPassword, setChangingPassword] = useState(false)
  const [logoutOpen, setLogoutOpen] = useState(false)
  const fileInputRef = useRef(null)
  const previewRef = useRef('')

  useEffect(() => {
    // eslint-disable-next-line react/set-state-in-effect -- sinkronisasi nama saat profil async (/me) selesai dimuat
    setName(user?.name ?? '')
  }, [user?.name])

  // Bebaskan object URL saat komponen dilepas.
  useEffect(() => {
    const ref = previewRef
    return () => {
      if (ref.current) URL.revokeObjectURL(ref.current)
    }
  }, [])

  function setPreviewUrl(url) {
    if (previewRef.current) URL.revokeObjectURL(previewRef.current)
    previewRef.current = url
    setPreview(url)
  }

  function handleLogout() {
    setLogoutOpen(false)
    logout()
    toast.info('Anda telah keluar dari aplikasi')
    navigate('/login', { replace: true })
  }

  const avatarSrc = preview || authFileUrl(user?.avatar_url)

  function handleFileChange(event) {
    const picked = event.target.files?.[0]
    if (!picked) return
    if (!/\.jpe?g$|\.png$|\.webp$/i.test(picked.name)) {
      toast.error('Avatar hanya boleh jpg/jpeg/png/webp.')
      return
    }
    if (
      picked.type &&
      !['image/jpeg', 'image/png', 'image/webp'].includes(picked.type)
    ) {
      toast.error('Isi file avatar harus gambar jpeg/png/webp.')
      return
    }
    if (picked.size > MAX_AVATAR_SIZE) {
      toast.error('Ukuran avatar maksimal 2MB.')
      return
    }
    setPreviewUrl(URL.createObjectURL(picked))
    setFile(picked)
  }

  async function handleSubmit(event) {
    event.preventDefault()
    if (!name.trim()) {
      toast.error('Nama wajib diisi.')
      return
    }
    setSaving(true)
    try {
      const formData = new FormData()
      formData.append('name', name.trim())
      if (file) formData.append('avatar', file)
      await authService.updateProfile(formData)
      await refreshUser()
      setPreviewUrl('')
      setFile(null)
      if (fileInputRef.current) fileInputRef.current.value = ''
      toast.success('Profil berhasil diperbarui!')
    } catch (err) {
      toast.error(apiErrorMessage(err, 'Gagal memperbarui profil.'))
    } finally {
      setSaving(false)
    }
  }

  async function handlePasswordSubmit(event) {
    event.preventDefault()
    if (!oldPassword || !newPassword) {
      toast.error('Password lama dan baru wajib diisi.')
      return
    }
    if (newPassword !== confirmPassword) {
      toast.error('Konfirmasi password tidak cocok.')
      return
    }
    setChangingPassword(true)
    try {
      await authService.changePassword({
        old_password: oldPassword,
        new_password: newPassword,
      })
      setOldPassword('')
      setNewPassword('')
      setConfirmPassword('')
      toast.success('Password berhasil diganti!')
    } catch (err) {
      toast.error(apiErrorMessage(err, 'Gagal mengganti password.'))
    } finally {
      setChangingPassword(false)
    }
  }

  return (
    <div className="min-h-screen bg-slate-50 pb-24 text-slate-800 md:pb-0 dark:bg-slate-950 dark:text-slate-100">
      <header className="border-b border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900">
        <div className="mx-auto flex max-w-3xl items-center justify-between gap-2 px-4 py-3">
          <div className="flex items-center gap-2">
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-emerald-600 text-white">
              <PiggyBank className="h-5 w-5" />
            </span>
            <span className="text-lg font-extrabold tracking-tight text-slate-900 dark:text-white">
              ArtaGoal
            </span>
          </div>
          <div className="flex items-center gap-2">
            <ThemeToggle />
            <Link
              to="/dashboard"
              className="flex min-h-[44px] items-center gap-1.5 rounded-lg border border-slate-200 px-4 py-2.5 text-sm font-medium text-slate-600 transition hover:border-emerald-300 hover:text-emerald-600 dark:border-slate-700 dark:text-slate-300 dark:hover:border-emerald-500 dark:hover:text-emerald-400"
            >
              <ArrowLeft className="h-4 w-4" /> Dashboard
            </Link>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-lg px-4 py-8">
        <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-800 dark:bg-slate-900 dark:shadow-lg">
          <h1 className="text-xl font-extrabold tracking-tight text-slate-900 dark:text-white">Profil Saya</h1>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
            Perbarui nama dan foto profilmu.
          </p>

          <form onSubmit={handleSubmit} className="mt-6 space-y-5">
            <div className="flex flex-col items-center">
              <div className="relative">
                {avatarSrc ? (
                  <img
                    src={avatarSrc}
                    alt="Foto profil"
                    className="h-28 w-28 rounded-full border-2 border-emerald-500 object-cover"
                  />
                ) : (
                  <div className="flex h-28 w-28 items-center justify-center rounded-full border-2 border-emerald-500 bg-emerald-100 text-3xl font-extrabold text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-400">
                    {initials(user?.name)}
                  </div>
                )}
                <button
                  type="button"
                  onClick={() => fileInputRef.current?.click()}
                  aria-label="Ubah foto profil"
                  className="absolute -right-1 -bottom-1 flex h-11 w-11 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow transition hover:border-emerald-300 hover:text-emerald-600 focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:border-emerald-500 dark:hover:text-emerald-400"
                >
                  <Camera className="h-4 w-4" />
                </button>
              </div>
              <input
                ref={fileInputRef}
                type="file"
                accept=".jpg,.jpeg,.png,.webp"
                onChange={handleFileChange}
                className="hidden"
              />
              <p className="mt-2 text-xs text-slate-400 dark:text-slate-500">
                JPG/PNG/WebP • maksimal 2MB
              </p>
            </div>

            <label className="block">
              <span className="mb-1.5 block text-sm font-semibold text-slate-700 dark:text-slate-300">
                Nama
              </span>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                maxLength={100}
                placeholder="Nama lengkap"
                className="min-h-[44px] w-full rounded-lg border border-slate-300 bg-white px-3 py-3 text-sm text-slate-900 outline-none transition placeholder:text-slate-400 focus:border-emerald-500 focus:ring-2 focus:ring-emerald-100 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100 dark:placeholder:text-slate-500 dark:focus:ring-emerald-500/20"
              />
            </label>

            <label className="block">
              <span className="mb-1.5 block text-sm font-semibold text-slate-700 dark:text-slate-300">
                Email
              </span>
              <input
                type="email"
                value={user?.email ?? ''}
                disabled
                className="min-h-[44px] w-full cursor-not-allowed rounded-lg border border-slate-200 bg-slate-100/70 px-3 py-3 text-sm text-slate-400 outline-none dark:border-slate-800 dark:bg-slate-800/50 dark:text-slate-500"
              />
            </label>

            <button
              type="submit"
              disabled={saving}
              className="flex min-h-[48px] w-full items-center justify-center gap-2 rounded-lg bg-emerald-600 px-4 py-3 text-sm font-semibold text-white transition hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {saving && <Loader2 className="h-4 w-4 animate-spin" />}
              {saving ? 'Menyimpan...' : 'Simpan Perubahan'}
            </button>
          </form>
        </div>

        <div className="mt-6 rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-800 dark:bg-slate-900 dark:shadow-lg">
          <h2 className="text-lg font-extrabold tracking-tight text-slate-900 dark:text-white">Keamanan</h2>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
            Ganti password akunmu secara berkala.
          </p>
          <form onSubmit={handlePasswordSubmit} className="mt-5 space-y-4">
            <label className="block">
              <span className="mb-1.5 block text-sm font-semibold text-slate-700 dark:text-slate-300">
                Password lama
              </span>
              <input
                type="password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                autoComplete="current-password"
                placeholder="••••••••"
                className="min-h-[44px] w-full rounded-lg border border-slate-300 bg-white px-3 py-3 text-sm text-slate-900 outline-none transition placeholder:text-slate-400 focus:border-emerald-500 focus:ring-2 focus:ring-emerald-100 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100 dark:placeholder:text-slate-500 dark:focus:ring-emerald-500/20"
              />
            </label>
            <div className="grid gap-4 sm:grid-cols-2">
              <label className="block">
                <span className="mb-1.5 block text-sm font-semibold text-slate-700 dark:text-slate-300">
                  Password baru
                </span>
                <input
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  autoComplete="new-password"
                  placeholder="Min. 8 karakter"
                  className="min-h-[44px] w-full rounded-lg border border-slate-300 bg-white px-3 py-3 text-sm text-slate-900 outline-none transition placeholder:text-slate-400 focus:border-emerald-500 focus:ring-2 focus:ring-emerald-100 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100 dark:placeholder:text-slate-500 dark:focus:ring-emerald-500/20"
                />
              </label>
              <label className="block">
                <span className="mb-1.5 block text-sm font-semibold text-slate-700 dark:text-slate-300">
                  Konfirmasi baru
                </span>
                <input
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  autoComplete="new-password"
                  placeholder="Ulangi password"
                  className="min-h-[44px] w-full rounded-lg border border-slate-300 bg-white px-3 py-3 text-sm text-slate-900 outline-none transition placeholder:text-slate-400 focus:border-emerald-500 focus:ring-2 focus:ring-emerald-100 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100 dark:placeholder:text-slate-500 dark:focus:ring-emerald-500/20"
                />
              </label>
            </div>
            <button
              type="submit"
              disabled={changingPassword}
              className="flex min-h-[48px] w-full items-center justify-center gap-2 rounded-lg border border-slate-300 px-4 py-3 text-sm font-semibold text-slate-700 transition hover:border-emerald-500 hover:text-emerald-600 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-700 dark:text-slate-100 dark:hover:border-emerald-500 dark:hover:text-emerald-400"
            >
              {changingPassword && <Loader2 className="h-4 w-4 animate-spin" />}
              {changingPassword ? 'Mengganti...' : 'Ganti Password'}
            </button>
          </form>
        </div>

        <button
          type="button"
          onClick={() => setLogoutOpen(true)}
          className="mt-6 flex min-h-[48px] w-full items-center justify-center gap-2 rounded-2xl border border-red-500/20 bg-red-500/10 px-4 py-3 text-sm font-semibold text-red-600 transition hover:bg-red-500/20 focus-visible:ring-2 focus-visible:ring-red-400 focus-visible:outline-none dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-300 dark:hover:bg-red-950/50"
        >
          <LogOut className="h-4 w-4" /> Keluar dari akun
        </button>

        <ConfirmModal
          isOpen={logoutOpen}
          onClose={() => setLogoutOpen(false)}
          onConfirm={handleLogout}
          title="Keluar dari ArtaGoal?"
          description="Apakah Anda yakin ingin keluar dari akun ArtaGoal?"
          confirmText="Ya, Keluar"
          variant="warning"
        />
      </main>
      <BottomNav />
    </div>
  )
}
