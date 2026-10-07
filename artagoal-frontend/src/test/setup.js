import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'
import '@testing-library/jest-dom/vitest'

// vitest berjalan tanpa globals sehingga auto-cleanup RTL tidak aktif;
// bersihkan DOM manual setiap test agar antar-test terisolasi.
afterEach(() => {
  cleanup()
})
