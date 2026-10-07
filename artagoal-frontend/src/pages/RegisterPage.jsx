import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { useAuth } from '../context/AuthContext.jsx'
import { apiErrorMessage } from '../services/api.js'
import { AuthShell, Field, SubmitButton } from '../components/AuthLayout.jsx'

export default function RegisterPage() {
  const { register } = useAuth()
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(event) {
    event.preventDefault()
    if (password !== confirm) {
      toast.error('Konfirmasi password tidak sama.')
      return
    }
    setLoading(true)
    try {
      await register(name.trim(), email.trim(), password)
      toast.success('Akun berhasil dibuat! Silakan masuk.')
      navigate('/dashboard', { replace: true })
    } catch (err) {
      toast.error(apiErrorMessage(err, 'Pendaftaran gagal. Coba lagi.'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <AuthShell
      title="Buat akun ArtaGoal"
      subtitle="Mulai rencanakan masa depan finansialmu hari ini."
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        <Field
          label="Nama lengkap"
          type="text"
          required
          autoComplete="name"
          placeholder="Nama kamu"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <Field
          label="Email"
          type="email"
          required
          autoComplete="email"
          placeholder="nama@email.com"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
        <Field
          label="Password (min. 8 karakter)"
          type="password"
          required
          minLength={8}
          autoComplete="new-password"
          placeholder="••••••••"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        <Field
          label="Konfirmasi password"
          type="password"
          required
          autoComplete="new-password"
          placeholder="••••••••"
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
        />
        <SubmitButton loading={loading}>Daftar</SubmitButton>
      </form>
      <p className="mt-5 text-center text-sm text-slate-500 dark:text-slate-400">
        Sudah punya akun?{' '}
        <Link to="/login" className="font-semibold text-emerald-700 hover:underline dark:text-emerald-400">
          Masuk
        </Link>
      </p>
    </AuthShell>
  )
}
