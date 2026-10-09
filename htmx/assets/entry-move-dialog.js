(() => {
  document.addEventListener("click", (event) => {
    const target = event.target;
    if (!(target instanceof Element)) {
      return;
    }
    const opener = target.closest("[data-entry-move]");
    if (opener) {
      const dialog = document.getElementById(opener.dataset.entryMove);
      if (!dialog || dialog.open) {
        return;
      }
      dialog.querySelector("form").reset();
      dialog.querySelector("[data-entry-move-confirm]").disabled = true;
      dialog.showModal();
      return;
    }
    const dismiss = target.closest("[data-entry-move-dismiss]");
    dismiss?.closest("dialog")?.close();
  });

  document.addEventListener("change", (event) => {
    const target = event.target;
    const form = target instanceof Element ? target.closest("[data-entry-move-form]") : null;
    if (form) {
      const destination = form.querySelector('input[name="destination"]:checked:not(:disabled)');
      form.querySelector("[data-entry-move-confirm]").disabled = !destination;
    }
  });

  document.addEventListener("submit", (event) => {
    const form = event.target;
    if (form instanceof HTMLFormElement && form.matches("[data-entry-move-form]")) {
      form.closest("dialog")?.close();
    }
  }, true);
})();
