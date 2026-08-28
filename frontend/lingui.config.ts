import type { LinguiConfig } from "@lingui/conf";
import { formatter } from "@lingui/format-po";
import { locales, sourceLocale } from "./src/i18n/locales";

const config: LinguiConfig = {
  locales: [...locales],
  sourceLocale,
  fallbackLocales: {
    default: sourceLocale,
  },
  catalogs: [
    {
      path: "<rootDir>/src/locales/{locale}",
      include: ["src"],
    },
  ],
  format: formatter({ lineNumbers: false }),
};

export default config;
