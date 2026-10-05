(() => {
  let draggedCard = null;
  let pending = false;

  function cardFromEvent(event) {
    const target = event.target;
    return target instanceof Element
      ? target.closest(".entry[data-entry-index][data-file]")
      : null;
  }

  function clearDropIndicators() {
    for (const card of document.querySelectorAll(".entry.drop-before, .entry.drop-after")) {
      card.classList.remove("drop-before", "drop-after");
    }
  }

  function clearDragState() {
    clearDropIndicators();
    draggedCard?.classList.remove("is-dragging");
    draggedCard = null;
  }

  document.addEventListener("dragstart", (event) => {
    const target = event.target;
    if (!(target instanceof Element)) {
      return;
    }
    const handle = target.closest("[data-entry-drag-handle]");
    const card = handle?.closest(".entry[data-entry-index][data-file]");
    if (!handle || !card || pending) {
      event.preventDefault();
      return;
    }

    draggedCard = card;
    card.classList.add("is-dragging");
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", card.dataset.entryIndex);
    const bounds = card.getBoundingClientRect();
    event.dataTransfer.setDragImage(card, Math.min(32, bounds.width / 2), 24);
  });

  document.addEventListener("dragover", (event) => {
    const targetCard = cardFromEvent(event);
    if (!draggedCard || pending || !targetCard || targetCard === draggedCard) {
      return;
    }

    event.preventDefault();
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = "move";
    }
    clearDropIndicators();
    const bounds = targetCard.getBoundingClientRect();
    targetCard.classList.add(event.clientY < bounds.top + bounds.height / 2 ? "drop-before" : "drop-after");
  });

  document.addEventListener("drop", async (event) => {
    if (!draggedCard || pending) {
      return;
    }
    event.preventDefault();

    const sourceCard = draggedCard;
    const targetCard = cardFromEvent(event);
    if (!targetCard || targetCard === sourceCard) {
      clearDragState();
      return;
    }

    const from = Number(sourceCard.dataset.entryIndex);
    const targetIndex = Number(targetCard.dataset.entryIndex);
    if (!Number.isInteger(from) || !Number.isInteger(targetIndex)) {
      clearDragState();
      return;
    }

    const bounds = targetCard.getBoundingClientRect();
    const before = event.clientY < bounds.top + bounds.height / 2;
    let to = targetIndex + (before ? 0 : 1);
    if (from < to) {
      to -= 1;
    }
    const cards = document.querySelectorAll("#workspace .entry[data-entry-index]");
    to = Math.max(0, Math.min(to, cards.length - 1));
    if (from === to) {
      clearDragState();
      return;
    }

    pending = true;
    document.getElementById("workspace")?.setAttribute("aria-busy", "true");
    clearDropIndicators();
    try {
      await window.htmx.ajax("POST", "/move", {
        target: "#workspace",
        swap: "outerHTML",
        values: {
          file: sourceCard.dataset.file,
          from: String(from),
          to: String(to),
        },
      });
    } catch (_) {
      window.location.reload();
    } finally {
      pending = false;
      document.getElementById("workspace")?.removeAttribute("aria-busy");
      clearDragState();
    }
  });

  document.addEventListener("dragend", () => {
    if (!pending) {
      clearDragState();
    }
  });
})();
