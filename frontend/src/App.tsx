import { Trans, useLingui } from "@lingui/react/macro";
import { useLayoutEffect, useState } from "react";
import { controls, heading, shell } from "./App.css";
import { ApplicationInfoPanel } from "./features/application-info/ApplicationInfoPanel";
import { activateLocale, defaultLocale, type Locale, locales } from "./i18n/i18n";
import { Button } from "./ui/components/Button/Button";
import { type ThemeName, themeClassNames, themeNames } from "./ui/tokens/themes.css";

/**
 * The foundation screen of the first production slice. It is deliberately not
 * the application shell: there is no main navigation and no save/character
 * panel yet.
 */
export function App() {
  const { t } = useLingui();
  const [theme, setTheme] = useState<ThemeName>("light");
  const [locale, setLocale] = useState<Locale>(defaultLocale);

  /**
   * The theme class lives on the document element, not on this screen, so
   * `html`, `body` and anything portalled under `body` resolve the same token
   * contract. The cleanup removes the previous class, so exactly one theme
   * class is ever active. It runs as a layout effect so the class is applied
   * before the browser paints and no frame is ever shown untokenised or in the
   * previous theme.
   */
  useLayoutEffect(() => {
    const root = document.documentElement;
    const themeClassName = themeClassNames[theme];
    root.classList.add(themeClassName);
    return () => root.classList.remove(themeClassName);
  }, [theme]);

  const themeLabels: Record<ThemeName, string> = {
    light: t`Light`,
    dark: t`Dark`,
    "elden-ring": t`Elden Ring`,
  };
  const localeLabels: Record<Locale, string> = {
    en: t`English`,
    pl: t`Polish`,
  };

  const switchLocale = async (next: Locale) => {
    await activateLocale(next);
    setLocale(next);
  };

  return (
    <main className={shell}>
      <h1 className={heading}>
        <Trans>SaveForge 2.0</Trans>
      </h1>

      <nav aria-label={t`Theme`} className={controls}>
        {themeNames.map((name) => (
          <Button key={name} pressed={theme === name} onClick={() => setTheme(name)}>
            {themeLabels[name]}
          </Button>
        ))}
      </nav>

      <nav aria-label={t`Language`} className={controls}>
        {locales.map((name) => (
          <Button key={name} pressed={locale === name} onClick={() => void switchLocale(name)}>
            {localeLabels[name]}
          </Button>
        ))}
      </nav>

      <ApplicationInfoPanel />
    </main>
  );
}
