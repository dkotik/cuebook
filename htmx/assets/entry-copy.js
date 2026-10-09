(() => {
  document.addEventListener("click", async (event) => {
    const target = event.target;
    const button = target instanceof Element ? target.closest("[data-entry-copy]") : null;
    if (!button || button.disabled) {
      return;
    }

    const source = document.getElementById(button.dataset.entryCopy);
    const status = button.closest(".entry-actions")?.querySelector("[data-entry-copy-status]");
    if (status) {
      status.textContent = "";
    }
    if (!source || !navigator.clipboard?.writeText) {
      window.htmx.trigger(document.body, "cuebook:flash", {
        message: "Clipboard access is unavailable. Use HTTPS or localhost to copy entries.",
      });
      return;
    }

    button.disabled = true;
    try {
      await navigator.clipboard.writeText(source.textContent);
      if (status) {
        status.textContent = "Copied";
      }
    } catch (_) {
      window.htmx.trigger(document.body, "cuebook:flash", {
        message: "The entry could not be copied. Please allow clipboard access and try again.",
      });
    } finally {
      button.disabled = false;
    }
  });
})();
