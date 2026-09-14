import i18n from "i18next"
import { initReactI18next } from "react-i18next"
import LanguageDetector from "i18next-browser-languagedetector"

import zhCN from "./locales/zh-CN.json"
import en from "./locales/en.json"

void i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      "zh-CN": { translation: zhCN },
      en: { translation: en },
    },
    fallbackLng: "zh-CN",
    supportedLngs: ["zh-CN", "en"],
    interpolation: { escapeValue: false },
    detection: {
      order: ["localStorage", "navigator"],
      caches: ["localStorage"],
      lookupLocalStorage: "nexa-locale",
    },
  })

export function syncDocumentLang(lng?: string): void {
  const lang = lng ?? i18n.language ?? "zh-CN"
  document.documentElement.lang = lang.startsWith("zh") ? "zh-CN" : "en"
}

i18n.on("languageChanged", syncDocumentLang)
syncDocumentLang()

export default i18n
