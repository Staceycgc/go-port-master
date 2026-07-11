const ALLOWED_AUTH_TYPES = new Set(['password', 'key'])

function asString(value) {
  return typeof value === 'string' ? value.trim() : ''
}

function asPort(value, fallback = 22) {
  const port = Number(value)
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    return fallback
  }
  return port
}

/**
 * Strip any credential fields; only persist safe host metadata.
 */
export function sanitizeRemoteHosts(hosts) {
  if (!Array.isArray(hosts)) {
    return []
  }
  const result = []
  for (const raw of hosts) {
    if (!raw || typeof raw !== 'object') {
      continue
    }
    const host = asString(raw.host)
    const username = asString(raw.username)
    if (!host || !username) {
      continue
    }
    const authType = ALLOWED_AUTH_TYPES.has(raw.authType) ? raw.authType : 'password'
    const id = asString(raw.id) || `host_${host}_${asPort(raw.port)}`
    const name = asString(raw.name) || host
    result.push({
      id,
      name,
      host,
      port: asPort(raw.port),
      username,
      authType
    })
  }
  return result
}

export function containsCredentialFields(hosts) {
  if (!Array.isArray(hosts)) {
    return false
  }
  const forbidden = ['credential', 'password', 'privateKey', 'private_key', 'passphrase', 'secret']
  return hosts.some((item) => {
    if (!item || typeof item !== 'object') {
      return false
    }
    return forbidden.some((key) => key in item && item[key] != null && item[key] !== '')
  })
}
