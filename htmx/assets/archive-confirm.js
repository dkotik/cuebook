(() => {
  const modal = document.getElementById("archive-confirmation");
  if (!modal) {
    return;
  }

  const entryLabel = modal.querySelector("[data-confirmation-entry]");
  const confirmButton = modal.querySelector("[data-confirmation-confirm]");
  const cancelButton = modal.querySelector(".archive-confirmation-cancel");
  let pendingForm = null;
  let pendingSubmitter = null;
  let pendingAction = null;
  let returnFocus = null;

  function close(restoreFocus) {
    modal.classList.remove("is-active");
    modal.setAttribute("aria-hidden", "true");
    document.documentElement.classList.remove("is-clipped");
    const focusTarget = returnFocus;
    pendingForm = null;
    pendingSubmitter = null;
    pendingAction = null;
    returnFocus = null;
    if (restoreFocus && focusTarget?.isConnected) {
      focusTarget.focus();
    }
  }

  function open(form, submitter) {
    pendingForm = form;
    pendingSubmitter = submitter instanceof HTMLElement ? submitter : null;
    pendingAction = null;
    returnFocus = pendingSubmitter;
    const title = form.closest(".entry")?.querySelector(".card-header-title")?.textContent?.trim();
    show(title);
  }

  function openDrop(sourceCard, confirm) {
    pendingForm = null;
    pendingSubmitter = null;
    pendingAction = confirm;
    returnFocus = sourceCard.querySelector("[data-entry-drag-handle]") || document.activeElement;
    const title = sourceCard.querySelector(".card-header-title")?.textContent?.trim();
    show(title);
  }

  function show(title) {
    if (entryLabel) {
      entryLabel.textContent = title ? `“${title}”` : "This entry";
    }
    modal.classList.add("is-active");
    modal.setAttribute("aria-hidden", "false");
    document.documentElement.classList.add("is-clipped");
    cancelButton?.focus();
  }

  document.addEventListener("submit", (event) => {
    const form = event.target;
    if (!(form instanceof HTMLFormElement) || !form.matches('form[action$="/archive"], form[hx-post$="/archive"]')) {
      return;
    }
    if (form.dataset.confirmed === "true") {
      delete form.dataset.confirmed;
      return;
    }

    event.preventDefault();
    event.stopImmediatePropagation();
    open(form, event.submitter);
  }, true);

  document.addEventListener("cuebook:archive-confirmation-request", (event) => {
    const { sourceCard, archiveTarget, confirm } = event.detail || {};
    if (!(sourceCard instanceof HTMLElement) || !(archiveTarget instanceof HTMLElement) ||
      !archiveTarget.matches("[data-archive-drop-target]") || sourceCard.dataset.canArchive !== "true" ||
      typeof confirm !== "function") {
      return;
    }

    event.preventDefault();
    if (pendingForm || pendingAction) {
      return;
    }
    openDrop(sourceCard, confirm);
  });

  modal.addEventListener("click", (event) => {
    if (event.target instanceof Element && event.target.closest("[data-confirmation-dismiss]")) {
      close(true);
    }
  });

  confirmButton?.addEventListener("click", async () => {
    const form = pendingForm;
    const submitter = pendingSubmitter;
    const action = pendingAction;
    close(false);
    if (typeof action === "function") {
      await action();
      return;
    }
    if (!form?.isConnected) {
      return;
    }

    form.dataset.confirmed = "true";
    try {
      if (submitter && form.contains(submitter)) {
        form.requestSubmit(submitter);
      } else {
        form.requestSubmit();
      }
    } finally {
      delete form.dataset.confirmed;
    }
  });

  modal.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      event.preventDefault();
      close(true);
      return;
    }
    if (event.key !== "Tab") {
      return;
    }

    const focusable = [...modal.querySelectorAll('button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])')];
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (!first || !last) {
      event.preventDefault();
    } else if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  });
})();
