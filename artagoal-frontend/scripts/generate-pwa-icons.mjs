// Generator ikon PWA ArtaGoal (tanpa dependensi, memakai zlib bawaan Node).
// Menghasilkan public/pwa-192x192.png dan public/pwa-512x512.png:
// latar slate-950 (#0f172a), badge emerald rounded-square, huruf "A" putih.
import { deflateSync } from 'node:zlib'
import { writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const NAVY = [15, 23, 42, 255]
const EMERALD = [5, 150, 105, 255]
const WHITE = [255, 255, 255, 255]

// Bitmap font 5x7 huruf "A".
const GLYPH_A = ['.###.', '#...#', '#...#', '#####', '#...#', '#...#', '#...#']

const crcTable = (() => {
  const table = new Uint32Array(256)
  for (let n = 0; n < 256; n += 1) {
    let c = n
    for (let k = 0; k < 8; k += 1) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1
    table[n] = c
  }
  return table
})()

function crc32(bytes) {
  let crc = 0xffffffff
  for (const b of bytes) crc = crcTable[(crc ^ b) & 0xff] ^ (crc >>> 8)
  return (crc ^ 0xffffffff) >>> 0
}

function chunk(type, data) {
  const len = Buffer.alloc(4)
  len.writeUInt32BE(data.length)
  const typeBuf = Buffer.from(type, 'ascii')
  const crc = Buffer.alloc(4)
  crc.writeUInt32BE(crc32(Buffer.concat([typeBuf, Buffer.from(data)])))
  return Buffer.concat([len, typeBuf, Buffer.from(data), crc])
}

function roundedRect(x, y, size, radius) {
  const cx = Math.min(Math.max(x, radius), size - radius)
  const cy = Math.min(Math.max(y, radius), size - radius)
  const dx = x - cx
  const dy = y - cy
  if (x >= radius && x < size - radius) return true
  if (y >= radius && y < size - radius) return true
  return dx * dx + dy * dy <= radius * radius
}

function render(size) {
  const px = Buffer.alloc(size * size * 4)
  const badgePad = Math.round(size * 0.14)
  const badgeR = Math.round(size * 0.16)
  const scale = Math.max(1, Math.floor((size * 0.34) / 7))
  const gw = 5 * scale
  const gh = 7 * scale
  const gx = Math.round((size - gw) / 2)
  const gy = Math.round((size - gh) / 2)

  for (let y = 0; y < size; y += 1) {
    for (let x = 0; x < size; x += 1) {
      let color = NAVY
      if (
        x >= badgePad &&
        x < size - badgePad &&
        y >= badgePad &&
        y < size - badgePad &&
        roundedRect(x - badgePad, y - badgePad, size - badgePad * 2, badgeR)
      ) {
        color = EMERALD
      }
      const lx = Math.floor((x - gx) / scale)
      const ly = Math.floor((y - gy) / scale)
      if (
        lx >= 0 &&
        lx < 5 &&
        ly >= 0 &&
        ly < 7 &&
        GLYPH_A[ly][lx] === '#'
      ) {
        color = WHITE
      }
      const i = (y * size + x) * 4
      px[i] = color[0]
      px[i + 1] = color[1]
      px[i + 2] = color[2]
      px[i + 3] = color[3]
    }
  }

  const raw = Buffer.alloc(size * (size * 4 + 1))
  for (let y = 0; y < size; y += 1) {
    raw[y * (size * 4 + 1)] = 0
    px.copy(raw, y * (size * 4 + 1) + 1, y * size * 4, (y + 1) * size * 4)
  }

  const ihdr = Buffer.alloc(13)
  ihdr.writeUInt32BE(size, 0)
  ihdr.writeUInt32BE(size, 4)
  ihdr[8] = 8 // bit depth
  ihdr[9] = 6 // color type RGBA
  const png = Buffer.concat([
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
    chunk('IHDR', ihdr),
    chunk('IDAT', deflateSync(raw)),
    chunk('IEND', Buffer.alloc(0)),
  ])
  const out = join(root, 'public', `pwa-${size}x${size}.png`)
  writeFileSync(out, png)
  console.log(`ditulis ${out} (${png.length} byte)`)
}

render(192)
render(512)
