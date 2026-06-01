/**
 * Parse an SSE text block into a single data payload string.
 * Handles multi-line data fields (joined by newlines).
 */
export const parseSSEBlock = (block: string): string =>
  block
    .split('\n')
    .map((line) => line.trimEnd())
    .filter((line) => line.startsWith('data:'))
    .map((line) => line.slice(5).trimStart())
    .join('\n')

/**
 * Split SSE buffer by double-newline boundary.
 * Handles both LF and CRLF line endings.
 */
export const splitSSEBlocks = (buffer: string): [string[], string] => {
  const blocks = buffer.split(/\r?\n\r?\n/)
  const remainder = blocks.pop() || ''
  return [blocks, remainder]
}
