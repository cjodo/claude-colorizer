// Port of internal/colors (find.go, color.go): finds color literals in text
// and converts them to 24-bit sRGB. Keep the two in step.

export type RGB = { r: number; g: number; b: number }

// A color literal found in text. start and end are string offsets.
export type Match = { start: number; end: number; text: string; color: RGB }

const hexRe = /#(?:[0-9a-fA-F]{8}|[0-9a-fA-F]{6}|[0-9a-fA-F]{3,4})/g
// rgb()/rgba()/hsl()/hsla()/oklch() with comma or space syntax and an
// optional alpha after "," or "/".
const fnRe =
  /\b(rgba?|hsla?|oklch)\(\s*([-+.\d]+(?:%|deg)?)\s*[,\s]\s*([-+.\d]+%?)\s*[,\s]\s*([-+.\d]+(?:%|deg)?)\s*(?:[,/]\s*[-+.\d]+%?\s*)?\)/gi

// find returns every color literal in s, in order of appearance.
export function find(s: string): Match[] {
  const out: Match[] = []
  for (const m of s.matchAll(hexRe)) {
    const start = m.index!
    const end = start + m[0].length
    if (!hexBoundary(s, start, end)) continue
    // "#123" is far more often an issue/PR number than a color, so short
    // forms must contain at least one hex letter.
    if (m[0].length <= 5 && !/[a-fA-F]/.test(m[0].slice(1))) continue
    const color = parseHex(m[0])
    if (color) out.push({ start, end, text: m[0], color })
  }
  for (const m of s.matchAll(fnRe)) {
    const [, name = '', x = '', y = '', z = ''] = m
    const color = parseFunc(name.toLowerCase(), [x, y, z])
    if (!color) continue
    out.push({ start: m.index!, end: m.index! + m[0].length, text: m[0], color })
  }
  return out.sort((a, b) => a.start - b.start)
}

// hexBoundary rejects hex runs embedded in longer tokens such as URL
// fragments (#section), HTML entities (&#123;) or longer hex strings.
function hexBoundary(s: string, start: number, end: number): boolean {
  const before = s.charAt(start - 1)
  const after = s.charAt(end)
  if (before === '&' || isWord(before)) return false
  if (isWord(after) || after === '-') return false
  return true
}

const isWord = (c: string) => /[A-Za-z0-9_]/.test(c)

export function parseHex(s: string): RGB | null {
  s = s.replace(/^#/, '')
  if (s.length === 3 || s.length === 4) s = [0, 1, 2].map(i => s.charAt(i).repeat(2)).join('')
  else if (s.length === 6 || s.length === 8) s = s.slice(0, 6)
  else return null
  if (!/^[0-9a-fA-F]{6}$/.test(s)) return null
  const v = parseInt(s, 16)
  return { r: (v >> 16) & 255, g: (v >> 8) & 255, b: v & 255 }
}

function parseFunc(name: string, a: [string, string, string]): RGB | null {
  const nx = num(a[0])
  const ny = num(a[1])
  const nz = num(a[2])
  if (!nx || !ny || !nz) return null
  const [[x, xPct], [y, yPct], [z, zPct]] = [nx, ny, nz]
  switch (name) {
    case 'rgb':
    case 'rgba': {
      const ch = (f: number, pct: boolean) => clamp8(pct ? (f * 255) / 100 : f)
      return { r: ch(x, xPct), g: ch(y, yPct), b: ch(z, zPct) }
    }
    case 'hsl':
    case 'hsla':
      return hslToRGB(x, y / 100, z / 100)
    case 'oklch': {
      const l = xPct || x > 1 ? x / 100 : x
      const c = yPct ? (y * 0.4) / 100 : y
      return oklchToRGB(l, c, z)
    }
  }
  return null
}

// num parses "12", "12.5%", "120deg" into [value, has trailing %].
function num(s: string): [number, boolean] | null {
  s = s.replace(/deg$/, '')
  const pct = s.endsWith('%')
  if (pct) s = s.slice(0, -1)
  const f = Number(s)
  return s === '' || Number.isNaN(f) ? null : [f, pct]
}

function clamp8(f: number): number {
  if (Number.isNaN(f) || f < 0) return 0
  if (f > 255) return 255
  return Math.round(f)
}

function hslToRGB(h: number, s: number, l: number): RGB {
  h = ((((h % 360) + 360) % 360) / 360)
  if (s === 0) {
    const v = clamp8(l * 255)
    return { r: v, g: v, b: v }
  }
  const q = l < 0.5 ? l * (1 + s) : l + s - l * s
  const p = 2 * l - q
  const hue = (t: number) => {
    if (t < 0) t++
    if (t > 1) t--
    if (t < 1 / 6) return p + (q - p) * 6 * t
    if (t < 1 / 2) return q
    if (t < 2 / 3) return p + (q - p) * (2 / 3 - t) * 6
    return p
  }
  return {
    r: clamp8(hue(h + 1 / 3) * 255),
    g: clamp8(hue(h) * 255),
    b: clamp8(hue(h - 1 / 3) * 255),
  }
}

// oklchToRGB converts OKLCH (L in [0,1], C, H in degrees) to gamut-clipped sRGB.
function oklchToRGB(l: number, c: number, h: number): RGB {
  const hr = (h * Math.PI) / 180
  const a = c * Math.cos(hr)
  const b = c * Math.sin(hr)

  const l_ = l + 0.3963377774 * a + 0.2158037573 * b
  const m_ = l - 0.1055613458 * a - 0.0638541728 * b
  const s_ = l - 0.0894841775 * a - 1.291485548 * b
  const l3 = l_ ** 3
  const m3 = m_ ** 3
  const s3 = s_ ** 3

  const r = 4.0767416621 * l3 - 3.3077115913 * m3 + 0.2309699292 * s3
  const g = -1.2684380046 * l3 + 2.6097574011 * m3 - 0.3413193965 * s3
  const bl = -0.0041960863 * l3 - 0.7034186147 * m3 + 1.707614701 * s3

  const gamma = (x: number) =>
    x <= 0.0031308 ? clamp8(12.92 * x * 255) : clamp8((1.055 * x ** (1 / 2.4) - 0.055) * 255)
  return { r: gamma(r), g: gamma(g), b: gamma(bl) }
}

export const hex = ({ r, g, b }: RGB) =>
  '#' + [r, g, b].map(v => v.toString(16).padStart(2, '0')).join('')

// contrast returns black or white, whichever reads better on c (WCAG).
export function contrast(c: RGB): string {
  const lin = (v: number) => {
    const f = v / 255
    return f <= 0.04045 ? f / 12.92 : ((f + 0.055) / 1.055) ** 2.4
  }
  const lum = 0.2126 * lin(c.r) + 0.7152 * lin(c.g) + 0.0722 * lin(c.b)
  return lum > 0.179 ? '#000000' : '#ffffff'
}
