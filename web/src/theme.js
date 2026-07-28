export const THEME_STORAGE_KEY = "cowyo-theme";
export const DARK_THEME_QUERY = "(prefers-color-scheme: dark)";

const themes = new Set(["light", "dark"]);
const themeColors = {
  light: "#f0f0f0",
  dark: "#171717",
};

export function readStoredTheme(storage) {
  try {
    const theme = storage.getItem(THEME_STORAGE_KEY);
    return themes.has(theme) ? theme : undefined;
  } catch {
    return undefined;
  }
}

export function storeTheme(storage, theme) {
  if (!themes.has(theme)) {
    return false;
  }

  try {
    storage.setItem(THEME_STORAGE_KEY, theme);
    return true;
  } catch {
    return false;
  }
}

export function systemTheme(colorSchemeQuery) {
  return colorSchemeQuery.matches ? "dark" : "light";
}

export function applyTheme(root, themeColorMeta, theme) {
  if (!themes.has(theme)) {
    return;
  }

  root.dataset.theme = theme;
  themeColorMeta?.setAttribute("content", themeColors[theme]);
}

export function applySystemTheme(root, themeColorMeta, colorSchemeQuery) {
  const theme = systemTheme(colorSchemeQuery);
  delete root.dataset.theme;
  themeColorMeta?.setAttribute("content", themeColors[theme]);
  return theme;
}
