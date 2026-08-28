/**
 * The single source of truth for the locales enabled in the current technical
 * foundation. It is not a decision about the language list of any release.
 * Both the runtime (`i18n.ts`) and the Lingui catalog configuration
 * (`lingui.config.ts`) read this list, so a locale can never be enabled in one
 * and missing in the other.
 *
 * English is the source locale and the safe fallback for every missing
 * translation. The Polish catalog is present to prove locale switching and the
 * English fallback.
 */
export const sourceLocale = "en";

export const locales = ["en", "pl"] as const;

export type Locale = (typeof locales)[number];
