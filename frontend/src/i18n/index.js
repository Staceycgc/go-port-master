import { createI18n } from 'vue-i18n'
import zhCN from './locales/zh-CN'
import en from './locales/en'
import { loadFromStorage, STORAGE_KEYS, getDefaultSettings } from '@/utils/storage'

function getInitialLocale() {
  const settings = loadFromStorage(STORAGE_KEYS.SETTINGS, getDefaultSettings())
  return settings.locale || 'zh-CN'
}

const i18n = createI18n({
  legacy: false,
  locale: getInitialLocale(),
  fallbackLocale: 'zh-CN',
  messages: {
    'zh-CN': zhCN,
    en
  }
})

export function applyDocumentLocale(locale = getLocale()) {
  const lang = locale === 'en' ? 'en' : 'zh-CN'
  document.documentElement.lang = lang
  document.title = i18n.global.t('app.documentTitle')
}

export function setLocale(locale) {
  i18n.global.locale.value = locale
  applyDocumentLocale(locale)
}

export function getLocale() {
  return i18n.global.locale.value
}

applyDocumentLocale(getInitialLocale())

export default i18n
