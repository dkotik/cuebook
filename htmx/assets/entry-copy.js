(() => {
  const cutKey = "cuebook-cut:" + (document.body?.dataset.routePrefix || "/");

  document.addEventListener("click", async (event) => {
    const target = event.target;
    const button = target instanceof Element ? target.closest("[data-entry-copy], [data-entry-cut]") : null;
    if (!button || button.disabled) {
      return;
    }

    const isCut = button.hasAttribute("data-entry-cut");
    const source = document.getElementById(button.dataset.entryCopy || button.dataset.entryCut);
    const actions = button.closest(".entry-actions");
    const status = actions?.querySelector("[data-entry-copy-status]");
    if (status) {
      status.textContent = "";
    }
    try {
      // Clear before writing so a failed copy or cut cannot retain an older move.
      localStorage.removeItem(cutKey);
    } catch (_) {
      window.htmx.trigger(document.body, "cuebook:flash", {
        message: "Unable to clear the cut status. Allow local storage before copying or cutting entries.",
      });
      return;
    }
    if (!source || !navigator.clipboard?.writeText) {
      window.htmx.trigger(document.body, "cuebook:flash", {
        message: "Clipboard access is unavailable. Use HTTPS or localhost to copy entries.",
      });
      return;
    }

    const buttons = actions.querySelectorAll("[data-entry-copy], [data-entry-cut]");
    buttons.forEach((control) => { control.disabled = true; });
    try {
      await navigator.clipboard.writeText(source.textContent);
      if (isCut) {
        try {
          localStorage.setItem(cutKey, JSON.stringify({
            file: button.dataset.file,
            entry: button.dataset.entryIndex,
            fingerprint: button.dataset.cutFingerprint,
          }));
        } catch (_) {
          window.htmx.trigger(document.body, "cuebook:flash", {
            message: "The entry was copied, but its cut status could not be saved. Pasting will copy it, not move it.",
          });
          return;
        }
      }
      if (status) {
        status.textContent = isCut ? "Cut — paste into another file to move" : "Copied";
      }
    } catch (_) {
      window.htmx.trigger(document.body, "cuebook:flash", {
        message: "The entry could not be copied. Please allow clipboard access and try again.",
      });
    } finally {
      buttons.forEach((control) => { control.disabled = false; });
    }
  });
})();
