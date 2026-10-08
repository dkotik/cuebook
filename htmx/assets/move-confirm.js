(() => {
  const modal = document.getElementById("move-confirmation");
  if (!modal) {
    return;
  }

  const entryLabel = modal.querySelector("[data-move-confirmation-entry]");
  const sourceLabel = modal.querySelector("[data-move-confirmation-source]");
  const destinationLabel = modal.querySelector("[data-move-confirmation-destination]");
  const confirmButton = modal.querySelector("[data-move-confirmation-confirm]");
  const cancelButton = modal.querySelector(".move-confirmation-cancel");
  let pendingMove = null;
  let returnFocus = null;

  function close(restoreFocus) {
    modal.classList.remove("is-active");
    modal.setAttribute("aria-hidden", "true");
    document.documentElement.classList.remove("is-clipped");
    const focusTarget = returnFocus;
    pendingMove = null;
    returnFocus = null;
    if (restoreFocus && focusTarget?.isConnected) {
      focusTarget.focus();
    }
  }

  function fileLabel(fileName) {
    const link = [...document.querySelectorAll(".tree-file-link[data-file]")]
      .find((link) => link.dataset.file === fileName);
    return link?.textContent?.trim() || fileName;
  }

  function open(sourceCard, destinationLink, confirmMove) {
    pendingMove = confirmMove;
    returnFocus = sourceCard.querySelector("[data-entry-drag-handle]") || document.activeElement;

    const title = sourceCard.querySelector(".card-header-title")?.textContent?.trim();
    if (entryLabel) {
      entryLabel.textContent = title ? `“${title}”` : "This entry";
    }
    if (sourceLabel) {
      sourceLabel.textContent = fileLabel(sourceCard.dataset.file);
    }
    if (destinationLabel) {
      destinationLabel.textContent = destinationLink.textContent.trim() || destinationLink.dataset.file;
    }

    modal.classList.add("is-active");
    modal.setAttribute("aria-hidden", "false");
    document.documentElement.classList.add("is-clipped");
    cancelButton?.focus();
  }

  document.addEventListener("cuebook:move-confirmation-request", (event) => {
    const { sourceCard, destinationLink, confirm } = event.detail || {};
    if (!(sourceCard instanceof HTMLElement) || !(destinationLink instanceof HTMLElement) || typeof confirm !== "function") {
      return;
    }
    if (!sourceCard.dataset.file || !destinationLink.dataset.file || sourceCard.dataset.file === destinationLink.dataset.file) {
      return;
    }

    event.preventDefault();
    if (pendingMove) {
      return;
    }
    open(sourceCard, destinationLink, confirm);
  });

  modal.addEventListener("click", (event) => {
    if (event.target instanceof Element && event.target.closest("[data-move-confirmation-dismiss]")) {
      close(true);
    }
  });

  confirmButton?.addEventListener("click", async () => {
    const confirmMove = pendingMove;
    close(false);
    if (typeof confirmMove === "function") {
      await confirmMove();
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
