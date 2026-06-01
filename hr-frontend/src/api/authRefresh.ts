type ClientApp = 'candidate' | 'hr'

export const silentRefresh = async (clientApp: ClientApp = 'hr'): Promise<void> => {
  const resp = await fetch(`${import.meta.env.VITE_API_BASE_URL || ''}/api/v1/auth/refresh`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'X-Client-App': clientApp },
  })
  const json = await resp.json()
  if (json.code !== 0) {
    throw new Error('refresh failed')
  }
}

