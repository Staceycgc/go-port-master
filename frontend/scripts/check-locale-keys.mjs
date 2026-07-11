import zhCN from '../src/i18n/locales/zh-CN.js'
import en from '../src/i18n/locales/en.js'

function flatten(obj, prefix = '') {
  const keys = []
  for (const [key, value] of Object.entries(obj)) {
    const path = prefix ? `${prefix}.${key}` : key
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      keys.push(...flatten(value, path))
    } else {
      keys.push(path)
    }
  }
  return keys.sort()
}

const zhKeys = flatten(zhCN)
const enKeys = flatten(en)
const zhSet = new Set(zhKeys)
const enSet = new Set(enKeys)

const missingInEn = zhKeys.filter((key) => !enSet.has(key))
const missingInZh = enKeys.filter((key) => !zhSet.has(key))

if (missingInEn.length || missingInZh.length) {
  if (missingInEn.length) {
    console.error('Missing in en.js:', missingInEn.join(', '))
  }
  if (missingInZh.length) {
    console.error('Missing in zh-CN.js:', missingInZh.join(', '))
  }
  process.exit(1)
}

console.log(`Locale keys aligned (${zhKeys.length} keys)`)
