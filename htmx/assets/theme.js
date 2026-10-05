(() => {
  const storageKey = "cuebook-theme";
  const root = document.documentElement;

  function loadTheme() {
    try {
      const saved = window.localStorage.getItem(storageKey);
      if (saved === "light" || saved === "dark") {
        return saved;
      }
    } catch (_) {
      // Storage can be unavailable in restricted browser contexts.
    }
    return "dark";
  }

  function updateToggle(theme) {
    const button = document.getElementById("theme-toggle");
    if (!button) {
      return;
    }
    button.setAttribute("aria-pressed", String(theme === "dark"));
    const label = button.querySelector("[data-theme-label]");
    if (label) {
      label.textContent = theme === "dark" ? "Dark mode" : "Light mode";
    }
  }

  function applyTheme(theme, persist) {
    root.setAttribute("data-theme", theme);
    updateToggle(theme);
    if (persist) {
      try {
        window.localStorage.setItem(storageKey, theme);
      } catch (_) {
        // Keep the in-memory theme even when the browser blocks storage.
      }
    }
  }

  applyTheme(loadTheme(), false);

  function initializeToggle() {
    const button = document.getElementById("theme-toggle");
    if (!button) {
      return;
    }
    updateToggle(root.getAttribute("data-theme") || "dark");
    button.addEventListener("click", () => {
      const current = root.getAttribute("data-theme");
      applyTheme(current === "dark" ? "light" : "dark", true);
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initializeToggle, { once: true });
  } else {
    initializeToggle();
  }
})();
