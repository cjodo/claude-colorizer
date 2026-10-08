// Inline color highlighting in Claude's replies, in the spirit of
// nvim-colorizer: each color literal is drawn on its own color.

import type { Register, RenderChildren } from 'claude-code'

import { contrast, hex } from './colors'
import { segment, type Segment } from './segment'

export const register: Register = on => {
  on('ui.render', { component: 'AssistantMessage' }, ($, e, next) => {
    if (e.props.isSummary) return next(e)

    const segments = segment(e.props.text)
    if (!segments.some(s => s.kind === 'line')) return next(e)

    const { Box, Text, Markdown } = $.ui.resolve(e)

    const draw = (s: Segment, i: number) => {
      switch (s.kind) {
        case 'markdown':
          return <Markdown key={`md${i}`} text={s.text} />
        case 'blank':
          return <Text key={`b${i}`}> </Text>
        case 'line': {
          const parts: RenderChildren[] = []
          let at = 0
          for (const m of s.matches) {
            if (m.start > at) parts.push(s.text.slice(at, m.start))
            parts.push(
              <Text backgroundColor={hex(m.color)} color={contrast(m.color)}>
                {m.text}
              </Text>,
            )
            at = m.end
          }
          if (at < s.text.length) parts.push(s.text.slice(at))
          return (
            <Box key={`l${i}`} paddingLeft={s.code ? 2 : 0}>
              <Text>{parts.length ? parts : ' '}</Text>
            </Box>
          )
        }
      }
    }

    // A tree of our own replaces the whole row, bullet included.
    return (
      <Box flexDirection="row">
        <Box width={2} flexShrink={0}>
          <Text color="text">{e.props.isFirstOfReply ? '●' : ' '}</Text>
        </Box>
        <Box flexDirection="column" flexGrow={1}>
          {segments.map(draw)}
        </Box>
      </Box>
    )
  })
}
