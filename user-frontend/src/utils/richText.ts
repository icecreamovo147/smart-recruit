const allowedTags = new Set(['p', 'br', 'strong', 'b', 'em', 'i', 'u', 'ul', 'ol', 'li', 'div', 'span', 'font'])
const fontSizeMap: Record<string, string> = {
  '1': '12px',
  '2': '13px',
  '3': '15px',
  '4': '17px',
  '5': '20px',
  '6': '24px',
  '7': '28px',
}

const escapeHtml = (value: string = ''): string =>
  String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')

const hasHtml = (value: string = ''): boolean => /<\/?[a-z][\s\S]*>/i.test(value)

const safeColor = (value: string = ''): string => {
  const color = String(value).trim()
  if (/^#[0-9a-f]{3}([0-9a-f]{3})?$/i.test(color)) return color
  if (/^rgb\(\s*\d{1,3}\s*,\s*\d{1,3}\s*,\s*\d{1,3}\s*\)$/i.test(color)) return color
  return ''
}

const safeStyle = (node: HTMLElement): string => {
  const color = safeColor(node.style?.color)
  const fontSize = /^(\d{1,2}px|1(\.\d)?em)$/.test(node.style?.fontSize || '') ? node.style.fontSize : ''
  const textAlign = /^(left|center|right)$/.test(node.style?.textAlign || '') ? node.style.textAlign : ''
  const styles: string[] = []
  if (color) styles.push(`color: ${color}`)
  if (fontSize) styles.push(`font-size: ${fontSize}`)
  if (textAlign) styles.push(`text-align: ${textAlign}`)
  return styles.join('; ')
}

export const renderRichText = (value: string = '', fallback: string = '暂无内容'): string => {
  const text = String(value || '').trim()
  if (!text) return `<p>${fallback}</p>`
  if (!hasHtml(text)) {
    return escapeHtml(text)
      .split(/\n{2,}/)
      .map((block) => `<p>${block.replace(/\n/g, '<br>')}</p>`)
      .join('')
  }
  const template = document.createElement('template')
  template.innerHTML = text
  template.content.querySelectorAll('*').forEach((node) => {
    const element = node as HTMLElement
    const tag = element.tagName.toLowerCase()
    if (!allowedTags.has(tag)) {
      element.replaceWith(document.createTextNode(element.textContent || ''))
      return
    }
    let nextStyle: string = ['span', 'p', 'div', 'li'].includes(tag) ? safeStyle(element) : ''
    if (tag === 'font') {
      const color = safeColor(element.getAttribute('color') || '')
      const size = fontSizeMap[element.getAttribute('size') || '']
      const styles: string[] = []
      if (color) styles.push(`color: ${color}`)
      if (size) styles.push(`font-size: ${size}`)
      nextStyle = styles.join('; ')
    }
    Array.from(element.attributes).forEach((attr) => element.removeAttribute(attr.name))
    if (nextStyle) element.setAttribute('style', nextStyle)
  })
  return template.innerHTML
}

export const plainTextSummary = (value: string = '', fallback: string = '暂无岗位描述'): string => {
  const template = document.createElement('template')
  template.innerHTML = hasHtml(value) ? value : escapeHtml(value)
  return (template.content.textContent || '').replace(/\s+/g, ' ').trim() || fallback
}
