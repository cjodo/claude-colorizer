// Splits a reply's markdown so lines holding color literals can be drawn as
// styled text while everything else keeps the engine's markdown renderer.

import { find, type Match } from './colors'

export type Segment =
  | { kind: 'markdown'; text: string }
  | { kind: 'blank' }
  // A line drawn as plain text with each literal on its own color; code is
  // true for a line from a fenced block.
  | { kind: 'line'; text: string; matches: Match[]; code: boolean }

const fenceRe = /^\s*(```|~~~)/

export function segment(text: string): Segment[] {
  const out: Segment[] = []
  let md: string[] = []

  const flush = () => {
    // Blank lines at a chunk's edges become spacers, so the gap between a
    // markdown chunk and a styled line matches the paragraph gap.
    let lead = 0
    while (lead < md.length && md[lead]?.trim() === '') lead++
    let tail = md.length
    while (tail > lead && md[tail - 1]?.trim() === '') tail--
    for (let i = 0; i < lead; i++) out.push({ kind: 'blank' })
    if (tail > lead) out.push({ kind: 'markdown', text: md.slice(lead, tail).join('\n') })
    for (let i = tail; i < md.length; i++) out.push({ kind: 'blank' })
    md = []
  }

  const lines = text.split('\n')
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i] ?? ''
    const fence = line.match(fenceRe)?.[1]
    if (fence) {
      let j = i + 1
      while (j < lines.length && !lines[j]?.trimStart().startsWith(fence)) j++
      const body = lines.slice(i + 1, j)
      const block = lines.slice(i, j + 1)
      i = j
      if (!body.some(l => find(l).length)) {
        md.push(...block)
        continue
      }
      flush()
      for (const l of body) out.push({ kind: 'line', text: l, matches: find(l), code: true })
      continue
    }

    if (!find(line).length) {
      md.push(line)
      continue
    }
    flush()
    const plain = stripInline(line)
    out.push({ kind: 'line', text: plain, matches: find(plain), code: false })
  }
  flush()
  return out
}

// stripInline drops the markdown syntax a styled line would otherwise show
// raw: emphasis and code markers, heading hashes, list bullets.
function stripInline(line: string): string {
  return line
    .replace(/^(\s*)#{1,6}\s+/, '$1')
    .replace(/^(\s*)[-*+]\s+/, '$1• ')
    .replace(/\*\*|__|`/g, '')
}
