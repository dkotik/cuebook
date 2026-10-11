(() => {
  let opener = null;
  let modalSavePending = false;

  const closeDialog = (modal) => {
    if (!modal?.classList.contains("is-active")) {
      return;
    }
    modal.classList.remove("is-active");
    modal.setAttribute("aria-hidden", "true");
    document.documentElement.classList.remove("is-clipped");
    modal.innerHTML = "";
    if (opener?.isConnected) {
      opener.focus();
    }
    opener = null;
  };

  document.addEventListener("htmx:beforeRequest", (event) => {
    const source = event.detail.elt;
    if (!(source instanceof Element)) {
      return;
    }
    if (source.hasAttribute("data-field-edit-trigger")) {
      opener = source;
    }
    if (source.closest("#field-edit-modal form")) {
      modalSavePending = true;
    }
  });

  document.addEventListener("htmx:afterRequest", (event) => {
    const source = event.detail.elt;
    const status = event.detail.xhr?.status;
    if (source instanceof Element && source.closest("#field-edit-modal form") && (status === 204 || status < 200 || status >= 300)) {
      modalSavePending = false;
    }
  });

  document.addEventListener("htmx:afterSwap", (event) => {
    const target = event.detail.target;
    if (!(target instanceof Element)) {
      return;
    }
    if (target.id === "workspace") {
      const status = event.detail.xhr?.status;
      if (modalSavePending && status >= 200 && status < 300 && status !== 204) {
        modalSavePending = false;
        closeDialog(document.getElementById("field-edit-modal"));
      }
      return;
    }
    if (target.id !== "field-edit-modal") {
      return;
    }
    target.classList.add("is-active");
    target.setAttribute("aria-hidden", "false");
    document.documentElement.classList.add("is-clipped");
    target.querySelector('input:not([type="hidden"]), textarea, select, button')?.focus();
  });

  document.addEventListener("click", (event) => {
    const target = event.target;
    if (!(target instanceof Element) || !target.closest("[data-field-edit-dismiss]")) {
      return;
    }
    closeDialog(document.getElementById("field-edit-modal"));
  });

  document.addEventListener("keydown", (event) => {
    const modal = document.getElementById("field-edit-modal");
    if (!modal?.classList.contains("is-active")) {
      return;
    }
    if (event.key === "Escape") {
      event.preventDefault();
      closeDialog(modal);
      return;
    }
    if (event.key !== "Tab") {
      return;
    }

    const focusable = [...modal.querySelectorAll('a[href], button:not([disabled]), input:not([type="hidden"]):not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])')];
    if (focusable.length === 0) {
      event.preventDefault();
      return;
    }
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && (document.activeElement === first || !modal.contains(document.activeElement))) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && (document.activeElement === last || !modal.contains(document.activeElement))) {
      event.preventDefault();
      first.focus();
    }
  });
})();
