import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import zhcn from './zh_CN/resource.json'
import en from './en/resource.json'

export const supportedLanguages = ['zh_CN', 'en'] as const
export type SupportedLanguage = (typeof supportedLanguages)[number]

const languageStorageKey = 'ai-disk-cleaner.language'

function isSupportedLanguage(language: string): language is SupportedLanguage {
  return supportedLanguages.includes(language as SupportedLanguage)
}

function normalizeLanguage(language: string): SupportedLanguage | undefined {
  const normalizedLanguage = language.toLowerCase().replace('_', '-')

  if (normalizedLanguage === 'zh' || normalizedLanguage.startsWith('zh-')) {
    return 'zh_CN'
  }
  if (normalizedLanguage === 'en' || normalizedLanguage.startsWith('en-')) {
    return 'en'
  }

  return undefined
}

function readSavedLanguage(): SupportedLanguage | undefined {
  try {
    const savedLanguage = window.localStorage.getItem(languageStorageKey)
    return savedLanguage && isSupportedLanguage(savedLanguage)
      ? savedLanguage
      : undefined
  } catch {
    return undefined
  }
}

function detectSystemLanguage(): SupportedLanguage {
  const systemLanguages = navigator.languages?.length
    ? navigator.languages
    : [navigator.language]

  for (const language of systemLanguages) {
    const supportedLanguage = normalizeLanguage(language)
    if (supportedLanguage) {
      return supportedLanguage
    }
  }

  return 'en'
}

function saveLanguage(language: SupportedLanguage) {
  try {
    window.localStorage.setItem(languageStorageKey, language)
  } catch {
    // The selected language still applies for this session when storage is blocked.
  }
}

const initialLanguage = readSavedLanguage() ?? detectSystemLanguage()
saveLanguage(initialLanguage)

i18n.use(initReactI18next).init(
  {
    resources: {
      zh_CN: { translation: zhcn },
      en: { translation: en },
    },
    lng: initialLanguage,
    fallbackLng: 'en',
    interpolation: {
      escapeValue: false,
    },
  },
  undefined,
)

export async function changeLanguage(language: SupportedLanguage) {
  saveLanguage(language)
  await i18n.changeLanguage(language)
}

export default i18n
