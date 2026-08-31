import { useLayoutEffect, useState } from "react";
import { AppShell } from "./features/app-shell/AppShell";
import { useSaveSessionFlow } from "./features/save-session/useSaveSessionFlow";
import { activateLocale, defaultLocale, type Locale } from "./i18n/i18n";
import { type ThemeName, themeClassNames } from "./ui/tokens/themes.css";

/**
 * The composition boundary of the production application shell. It owns only
 * global presentation preferences and one session controller, then passes both
 * to AppShell. No screen creates a second copy of session state.
 */
export function App() {
  const [theme, setTheme] = useState<ThemeName>("light");
  const [locale, setLocale] = useState<Locale>(defaultLocale);
  const flow = useSaveSessionFlow();

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

  const switchLocale = async (next: Locale) => {
    await activateLocale(next);
    setLocale(next);
  };

  return (
    <AppShell
      flow={flow}
      theme={theme}
      onThemeChange={setTheme}
      locale={locale}
      onLocaleChange={(next) => void switchLocale(next)}
    />
  );
}
