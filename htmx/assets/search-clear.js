(() => {
  document.addEventListener("click", (event) => {
    const target = event.target;
    if (!(target instanceof Element)) {
      return;
    }

    const button = target.closest("[data-clear-search]");
    if (!button) {
      return;
    }

    const input = button.closest("form")?.querySelector('input[type="search"][name="q"]');
    if (!(input instanceof HTMLInputElement)) {
      return;
    }

    input.value = "";
    input.focus();
  });
})();
