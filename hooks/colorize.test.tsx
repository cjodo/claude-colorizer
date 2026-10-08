import { describe, expect, test } from 'claude-code/testing'

import { find } from './colors'
import { segment } from './segment'

const SURFACES = ['terminal', 'desktop'] as const

const row = (text: string) => ({
  plugin: 'claude-colorizer',
  component: 'AssistantMessage' as const,
  props: { text, isFirstOfReply: true },
})

describe('find', () => {
  test('parses every literal form', async () => {
    const hexes = find('a #f80 b #3b82f680 c rgb(255, 99, 71) d hsl(0 100% 50%) e oklch(0.7 0.15 200)')
    expect(hexes.map(m => m.text)).toEqual([
      '#f80',
      '#3b82f680',
      'rgb(255, 99, 71)',
      'hsl(0 100% 50%)',
      'oklch(0.7 0.15 200)',
    ])
    expect(hexes[2]?.color).toEqual({ r: 255, g: 99, b: 71 })
    expect(hexes[3]?.color).toEqual({ r: 255, g: 0, b: 0 })
  })

  test('skips issue numbers, anchors and entities', async () => {
    expect(find('fixes #123, see page#fff and &#abc;')).toEqual([])
  })
})

describe('segment', () => {
  test('keeps colorless fences as markdown and splits colored ones', async () => {
    const s = segment('intro\n\n```css\na { color: red; }\n```\n\n```css\nb { color: #f38ba8; }\n```')
    expect(s.map(x => x.kind)).toEqual(['markdown', 'blank', 'line'])
    expect(s[2]).toMatchObject({ code: true, text: 'b { color: #f38ba8; }' })
  })

  test('strips inline markdown from colored lines', async () => {
    expect(segment('- **Rose** `#f38ba8`')[0]).toMatchObject({ text: '• Rose #f38ba8' })
  })
})

test('a reply without colors is drawn by the engine', async ($, on) => {
  on('ui.render', ($, e) => {
    const { Text } = $.ui.resolve(e)
    return <Text>engine</Text>
  })
  for (const surface of SURFACES) {
    const ui = await $.ui.mount({ ...row('No colors here.'), surface })
    expect(await ui.find({ type: 'Text', text: 'engine' })).toBeDefined()
    await ui.unmount()
  }
})

test('each literal is drawn on its own color', async $ => {
  for (const surface of SURFACES) {
    const ui = await $.ui.mount({ ...row('Some text.\n\nRose `#f38ba8` and white #fff.'), surface })
    expect(await ui.find({ type: 'Markdown', text: 'Some text.' })).toBeDefined()
    const rose = await ui.find({ type: 'Text', text: /^#f38ba8$/ })
    expect(rose?.props).toMatchObject({ backgroundColor: '#f38ba8', color: '#000000' })
    const white = await ui.find({ type: 'Text', text: /^#fff$/ })
    expect(white?.props).toMatchObject({ backgroundColor: '#ffffff', color: '#000000' })
    await ui.unmount()
  }
})
