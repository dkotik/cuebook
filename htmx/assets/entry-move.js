(() => {
  const routePrefix = document.body?.dataset.routePrefix || "";
  let draggedCard = null;
  let pending = false;

  function cardFromEvent(event) {
    const target = event.target;
    return target instanceof Element
      ? target.closest(".entry[data-entry-index][data-file]")
      : null;
  }

  function fileLinkFromEvent(event) {
    const target = event.target;
    return target instanceof Element ? target.closest(".tree-file-link[data-file]") : null;
  }

  function archiveTargetFromEvent(event) {
    const target = event.target;
    return target instanceof Element ? target.closest("[data-archive-drop-target]") : null;
  }

  function clearDropIndicators() {
    for (const card of document.querySelectorAll(
      ".entry.drop-before, .entry.drop-after, .entry.drop-above, .entry.drop-below, .entry.drop-left, .entry.drop-right",
    )) {
      card.classList.remove("drop-before", "drop-after", "drop-above", "drop-below", "drop-left", "drop-right");
    }
    for (const link of document.querySelectorAll(".tree-file-link.is-drop-target, [data-archive-drop-target].is-drop-target")) {
      link.classList.remove("is-drop-target");
    }
  }

  function dropSide(event, bounds) {
    const x = Math.min(1, Math.max(0, (event.clientX - bounds.left) / (bounds.width || 1)));
    const y = Math.min(1, Math.max(0, (event.clientY - bounds.top) / (bounds.height || 1)));
    const edges = [
      { side: "drop-left", distance: x },
      { side: "drop-right", distance: 1 - x },
      { side: "drop-above", distance: y },
      { side: "drop-below", distance: 1 - y },
    ];
    return edges.reduce((nearest, edge) => edge.distance < nearest.distance ? edge : nearest).side;
  }

  function clearDragState() {
    clearDropIndicators();
    draggedCard?.classList.remove("is-dragging");
    draggedCard = null;
  }

  function moveRequest(sourceCard, changes = {}) {
    const values = { file: sourceCard.dataset.file };
    const uuid = sourceCard.dataset.entryUuid;
    const path = uuid ? "/move/uuid" : "/move";
    if (uuid) {
      values.from_uuid = uuid;
    } else {
      values.from = sourceCard.dataset.entryIndex;
    }
    return { url: `${routePrefix}${path}`, values: { ...values, ...changes } };
  }

  function archiveRequest(sourceCard) {
    const values = { file: sourceCard.dataset.file };
    const uuid = sourceCard.dataset.entryUuid;
    const path = uuid ? "/archive/uuid" : "/archive";
    if (uuid) {
      values.uuid = uuid;
    } else {
      values.entry = sourceCard.dataset.entryIndex;
    }
    return { url: `${routePrefix}${path}`, values };
  }

  async function submitMove(sourceCard, request) {
    pending = true;
    document.getElementById("workspace")?.setAttribute("aria-busy", "true");
    clearDropIndicators();
    try {
      await window.htmx.ajax("POST", request.url, {
        target: "#workspace",
        swap: "outerHTML",
        values: request.values,
      });
    } catch (_) {
      window.location.reload();
    } finally {
      pending = false;
      document.getElementById("workspace")?.removeAttribute("aria-busy");
      sourceCard.classList.remove("is-dragging");
      clearDragState();
    }
  }

  async function submitArchive(sourceCard, request) {
    pending = true;
    document.getElementById("workspace")?.setAttribute("aria-busy", "true");
    clearDropIndicators();
    try {
      await window.htmx.ajax("POST", request.url, {
        target: "#workspace",
        swap: "outerHTML",
        values: request.values,
      });
    } catch (_) {
      window.location.reload();
    } finally {
      pending = false;
      document.getElementById("workspace")?.removeAttribute("aria-busy");
      sourceCard.classList.remove("is-dragging");
      clearDragState();
    }
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
    if (!draggedCard || pending) {
      return;
    }

    const targetArchive = archiveTargetFromEvent(event);
    if (targetArchive) {
      clearDropIndicators();
      if (draggedCard.dataset.canArchive === "true") {
        event.preventDefault();
        targetArchive.classList.add("is-drop-target");
        if (event.dataTransfer) {
          event.dataTransfer.dropEffect = "move";
        }
      }
      return;
    }

    const targetFileLink = fileLinkFromEvent(event);
    if (targetFileLink) {
      event.preventDefault();
      clearDropIndicators();
      if (targetFileLink.dataset.file !== draggedCard.dataset.file) {
        targetFileLink.classList.add("is-drop-target");
      }
      if (event.dataTransfer) {
        event.dataTransfer.dropEffect = "move";
      }
      return;
    }

    const targetCard = cardFromEvent(event);
    if (!targetCard || targetCard === draggedCard || targetCard.dataset.file !== draggedCard.dataset.file) {
      clearDropIndicators();
      return;
    }

    event.preventDefault();
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = "move";
    }
    clearDropIndicators();
    const side = dropSide(event, targetCard.getBoundingClientRect());
    const before = side === "drop-left" || side === "drop-above";
    targetCard.classList.add(side, before ? "drop-before" : "drop-after");
  });

  document.addEventListener("drop", async (event) => {
    if (!draggedCard || pending) {
      return;
    }
    event.preventDefault();

    const sourceCard = draggedCard;
    const targetArchive = archiveTargetFromEvent(event);
    if (targetArchive) {
      if (sourceCard.dataset.canArchive === "true") {
        const request = archiveRequest(sourceCard);
        document.dispatchEvent(new CustomEvent("cuebook:archive-confirmation-request", {
          cancelable: true,
          detail: {
            sourceCard,
            archiveTarget: targetArchive,
            confirm: () => submitArchive(sourceCard, request),
          },
        }));
      }
      clearDragState();
      return;
    }

    const targetFileLink = fileLinkFromEvent(event);
    if (targetFileLink) {
      const destination = targetFileLink.dataset.file;
      if (destination && destination !== sourceCard.dataset.file) {
        const move = moveRequest(sourceCard, { destination });
        const request = new CustomEvent("cuebook:move-confirmation-request", {
          cancelable: true,
          detail: {
            sourceCard,
            destinationLink: targetFileLink,
            confirm: () => submitMove(sourceCard, move),
          },
        });
        if (document.dispatchEvent(request)) {
          await submitMove(sourceCard, move);
        }
      } else {
        clearDragState();
      }
      return;
    }

    const targetCard = cardFromEvent(event);
    if (!targetCard || targetCard === sourceCard || targetCard.dataset.file !== sourceCard.dataset.file) {
      clearDragState();
      return;
    }

    const from = Number(sourceCard.dataset.entryIndex);
    const targetIndex = Number(targetCard.dataset.entryIndex);
    if (!Number.isInteger(from) || !Number.isInteger(targetIndex)) {
      clearDragState();
      return;
    }

    const side = dropSide(event, targetCard.getBoundingClientRect());
    const before = side === "drop-left" || side === "drop-above";
    let to = targetIndex + (before ? 0 : 1);
    if (from < to) {
      to -= 1;
    }
    const sourceEntryCount = Number(sourceCard.dataset.entryCount);
    const entryCount = Number.isInteger(sourceEntryCount) && sourceEntryCount > 0
      ? sourceEntryCount
      : document.querySelectorAll("#workspace .entry[data-entry-index]").length;
    to = Math.max(0, Math.min(to, entryCount - 1));
    if (from === to) {
      clearDragState();
      return;
    }

    await submitMove(sourceCard, moveRequest(sourceCard, { to: String(to) }));
  });

  document.addEventListener("dragend", () => {
    if (!pending) {
      clearDragState();
    }
  });
})();
