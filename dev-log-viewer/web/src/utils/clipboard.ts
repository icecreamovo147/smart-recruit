export interface ClipboardLike {
  writeText(text: string): Promise<void>
}

export async function copyText(text: string, clipboard: ClipboardLike | undefined = navigator.clipboard): Promise<void> {
  if (!clipboard) {
    throw new Error('Clipboard API is unavailable')
  }
  await clipboard.writeText(text)
}
