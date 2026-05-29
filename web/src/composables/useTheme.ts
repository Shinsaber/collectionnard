import { ref, watch } from "vue";

type Theme = "light" | "dark";

const STORAGE_KEY = "colec-theme";

function getInitialTheme(): Theme {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === "dark" || stored === "light") return stored;
  return window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

const theme = ref<Theme>(getInitialTheme());

function applyTheme(t: Theme) {
  document.documentElement.classList.toggle("dark", t === "dark");
}

// Apply on load
applyTheme(theme.value);

watch(theme, (t) => {
  localStorage.setItem(STORAGE_KEY, t);
  applyTheme(t);
});

export function useTheme() {
  function toggle() {
    theme.value = theme.value === "dark" ? "light" : "dark";
  }

  return { theme, toggle };
}
