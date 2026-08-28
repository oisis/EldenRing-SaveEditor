import { i18n } from "@lingui/core";
import { type Locale, sourceLocale } from "./locales";

export type { Locale } from "./locales";
export { locales, sourceLocale } from "./locales";

/** English is both the source locale and the safe fallback. */
export const defaultLocale: Locale = sourceLocale;

/**
 * Loads and activates a message catalog. Catalogs are compiled from the `.po`
 * sources by the Lingui Vite plugin, so the `fallbackLocales` configured in
 * `lingui.config.ts` fill every missing translation with the English source
 * string instead of showing an empty label or a raw message id.
 *
 * The semantic document language follows the active catalog, so assistive
 * technology and the browser hyphenation rules match what is rendered.
 */
export async function activateLocale(locale: Locale): Promise<void> {
  const { messages } = await import(`../locales/${locale}.po`);
  i18n.loadAndActivate({ locale, messages });
  document.documentElement.lang = locale;
}

export { i18n };
