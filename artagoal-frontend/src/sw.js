// Service worker ArtaGoal (strategi injectManifest via vite-plugin-pwa).
// Menangani precache hasil build + push notification dari backend.
import { cleanupOutdatedCaches, precacheAndRoute } from 'workbox-precaching'

cleanupOutdatedCaches()
precacheAndRoute(self.__WB_MANIFEST)

self.addEventListener('push', (event) => {
  let data = {}
  try {
    data = event.data ? event.data.json() : {}
  } catch {
    data = {}
  }
  const title = data.title || 'ArtaGoal'
  const options = {
    body: data.body || 'Ada kabar dari target keuanganmu.',
    icon: '/pwa-192x192.png',
    badge: '/pwa-192x192.png',
    data: { url: data.url || '/' },
  }
  event.waitUntil(self.registration.showNotification(title, options))
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const url = event.notification?.data?.url || '/'
  event.waitUntil(
    (async () => {
      const windows = await self.clients.matchAll({
        type: 'window',
        includeUncontrolled: true,
      })
      for (const client of windows) {
        if ('focus' in client) {
          await client.focus()
          if ('navigate' in client && client.url !== url) {
            await client.navigate(url)
          }
          return
        }
      }
      if (self.clients.openWindow) {
        await self.clients.openWindow(url)
      }
    })(),
  )
})
