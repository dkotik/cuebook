(() => {
  const selector = "textarea[data-markdown-editor]";
  const editors = new WeakMap();

  const textareasWithin = (root) => {
    if (!root) return [];
    const textareas = [];
    if (root.matches?.(selector)) textareas.push(root);
    if (root.querySelectorAll) textareas.push(...root.querySelectorAll(selector));
    return textareas;
  };

  const initializeWithin = (root = document) => {
    if (typeof window.EasyMDE !== "function" || !window.DOMPurify) return;

    for (const textarea of textareasWithin(root)) {
      if (editors.has(textarea)) continue;
      const editor = new window.EasyMDE({
        element: textarea,
        renderingConfig: {
          sanitizerFunction: (html) => window.DOMPurify.sanitize(html),
        },
      });
      editor.codemirror.on("change", () => {
        textarea.value = editor.value();
      });
      editors.set(textarea, editor);
    }
  };

  const destroyWithin = (root) => {
    for (const textarea of textareasWithin(root)) {
      const editor = editors.get(textarea);
      if (!editor) continue;
      textarea.value = editor.value();
      editor.toTextArea();
      editors.delete(textarea);
    }
  };

  document.addEventListener("submit", (event) => {
    for (const textarea of textareasWithin(event.target)) {
      const editor = editors.get(textarea);
      if (editor) textarea.value = editor.value();
    }
  }, true);

  document.addEventListener("htmx:afterSwap", (event) => {
    initializeWithin(event.detail?.target);
  });

  document.addEventListener("htmx:beforeCleanupElement", (event) => {
    destroyWithin(event.detail?.elt);
  });

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", () => initializeWithin(document), { once: true });
  } else {
    initializeWithin(document);
  }

  window.CuebookMarkdownEditor = Object.freeze({ initializeWithin, destroyWithin });
})();
