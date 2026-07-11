/**
 * LocalStorage 持久化工具
 */
const STORAGE_KEYS = {
  GROUPS: 'portmaster_groups',
  MONITOR: 'portmaster_monitor',
  HISTORY: 'portmaster_history',
  SETTINGS: 'portmaster_settings',
  SCAN_HISTORY: 'portmaster_scan_history',
  REMOTE_HOSTS: 'portmaster_remote_hosts'
}

/** 默认设置 */
export function getDefaultSettings() {
  return {
    locale: 'zh-CN',
    theme: 'light',
    autoRefreshInterval: 0,
    listenOnly: false,
    defaultPageSize: 50
  }
}

export function loadFromStorage(key, defaultValue = null) {
  try {
    const raw = localStorage.getItem(key)
    return raw ? JSON.parse(raw) : defaultValue
  } catch {
    return defaultValue
  }
}

export function saveToStorage(key, value) {
  localStorage.setItem(key, JSON.stringify(value))
}

export { STORAGE_KEYS }

/** 默认端口分组 */
export function getDefaultGroups(locale = 'zh-CN') {
  if (locale === 'en') {
    return [
      { id: 'web', name: 'Web Services', ports: [{ port: 8080, remark: '8080-default' }, { port: 8848, remark: '8848-Nacos' }] },
      { id: 'frontend', name: 'Frontend', ports: [{ port: 3000, remark: '3000-React' }, { port: 5173, remark: '5173-Vite' }] },
      { id: 'database', name: 'Databases', ports: [{ port: 3306, remark: '3306-MySQL' }, { port: 6379, remark: '6379-Redis' }, { port: 5432, remark: '5432-PostgreSQL' }] },
      { id: 'middleware', name: 'Middleware', ports: [{ port: 9092, remark: '9092-Kafka' }, { port: 5672, remark: '5672-RabbitMQ' }, { port: 9200, remark: '9200-Elasticsearch' }] }
    ]
  }
  return [
    { id: 'web', name: 'Web 服务', ports: [{ port: 8080, remark: '8080-默认服务' }, { port: 8848, remark: '8848-Nacos' }] },
    { id: 'frontend', name: '前端', ports: [{ port: 3000, remark: '3000-React' }, { port: 5173, remark: '5173-Vite' }] },
    { id: 'database', name: '数据库', ports: [{ port: 3306, remark: '3306-MySQL' }, { port: 6379, remark: '6379-Redis' }, { port: 5432, remark: '5432-PostgreSQL' }] },
    { id: 'middleware', name: '中间件', ports: [{ port: 9092, remark: '9092-Kafka' }, { port: 5672, remark: '5672-RabbitMQ' }, { port: 9200, remark: '9200-Elasticsearch' }] }
  ]
}

/** 内置常用端口库 */
export const COMMON_PORTS = [
  { name: 'MySQL', port: 3306, protocol: 'TCP' },
  { name: 'Redis', port: 6379, protocol: 'TCP' },
  { name: 'Nginx', port: 80, protocol: 'TCP' },
  { name: 'Nginx SSL', port: 443, protocol: 'TCP' },
  { name: 'Nacos', port: 8848, protocol: 'TCP' },
  { name: 'Tomcat', port: 8080, protocol: 'TCP' },
  { name: 'HTTP App', port: 8080, protocol: 'TCP' },
  { name: 'Node.js', port: 3000, protocol: 'TCP' },
  { name: 'MongoDB', port: 27017, protocol: 'TCP' },
  { name: 'PostgreSQL', port: 5432, protocol: 'TCP' },
  { name: 'RabbitMQ', port: 5672, protocol: 'TCP' },
  { name: 'Kafka', port: 9092, protocol: 'TCP' },
  { name: 'Elasticsearch', port: 9200, protocol: 'TCP' },
  { name: 'SSH', port: 22, protocol: 'TCP' },
  { name: 'Docker', port: 2375, protocol: 'TCP' }
]

/** 空闲端口段模板 */
export const PORT_TEMPLATES = [
  { labelKey: 'freePort.template80xx', start: 8000, count: 10 },
  { labelKey: 'freePort.template30xx', start: 3000, count: 10 },
  { labelKey: 'freePort.template90xx', start: 9000, count: 10 }
]
