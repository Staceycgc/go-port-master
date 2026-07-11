/**
 * WebSocket monitor client with token authentication.
 */
import { getAuthToken } from '@/utils/auth'

let ws = null
let reconnectTimer = null
let pingTimer = null
let alertCallback = null
let statusCallback = null

function getWsUrl() {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  const token = getAuthToken()
  const query = token ? `?token=${encodeURIComponent(token)}` : ''
  return `${proto}//${host}/ws/monitor${query}`
}

function notifyStatus(state) {
  statusCallback?.(state)
}

export function connectMonitorWs(onAlert, onStatus) {
  alertCallback = onAlert
  if (onStatus) {
    statusCallback = onStatus
  }
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
    notifyStatus(ws.readyState === WebSocket.OPEN ? 'open' : 'connecting')
    return
  }

  try {
    notifyStatus('connecting')
    ws = new WebSocket(getWsUrl())

    ws.onopen = () => {
      notifyStatus('open')
      startPing()
    }

    ws.onmessage = (event) => {
      if (event.data === 'pong') return
      try {
        const data = JSON.parse(event.data)
        if (data.type === 'alert' && data.alerts && alertCallback) {
          alertCallback(data.alerts)
        }
      } catch { /* ignore */ }
    }

    ws.onclose = () => {
      notifyStatus('closed')
      stopPing()
      scheduleReconnect()
    }

    ws.onerror = () => {
      notifyStatus('closed')
      ws?.close()
    }
  } catch {
    notifyStatus('closed')
    scheduleReconnect()
  }
}

export function disconnectMonitorWs() {
  stopPing()
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  notifyStatus('closed')
  if (ws) {
    ws.onclose = null
    ws.close()
    ws = null
  }
  alertCallback = null
  statusCallback = null
}

function scheduleReconnect() {
  if (reconnectTimer) return
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null
    if (alertCallback) connectMonitorWs(alertCallback, statusCallback)
  }, 5000)
}

function startPing() {
  stopPing()
  pingTimer = setInterval(() => {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send('ping')
    }
  }, 30000)
}

function stopPing() {
  if (pingTimer) {
    clearInterval(pingTimer)
    pingTimer = null
  }
}

/** Sync monitor config to server for background polling. */
export async function syncMonitorConfig(request, config) {
  await request.post('/monitor/config', {
    enabled: config.enabled,
    ports: (config.ports || []).map(p => ({
      port: p.port,
      protocol: p.protocol || 'TCP',
      remark: p.remark,
      expectedState: p.expectedState || 'any'
    }))
  })
}

export function isMonitorWsConnected() {
  return ws?.readyState === WebSocket.OPEN
}

export function getMonitorWsStatus() {
  if (!ws) return 'closed'
  switch (ws.readyState) {
    case WebSocket.CONNECTING:
      return 'connecting'
    case WebSocket.OPEN:
      return 'open'
    default:
      return 'closed'
  }
}
