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

  const button = (label, title, action) => {
    const control = document.createElement("button");
    control.className = "markdown-editor-button";
    control.type = "button";
    control.textContent = label;
    control.title = title;
    control.setAttribute("aria-label", title);
    control.addEventListener("click", action);
    return control;
  };

  const toolbar = (editor) => {
    const controls = document.createElement("div");
    controls.className = "markdown-editor-toolbar";
    controls.setAttribute("role", "toolbar");
    controls.setAttribute("aria-label", "Markdown formatting");

    const command = (label, title, run) => button(label, title, () => run(editor.chain().focus()).run());
    controls.append(
      command("B", "Bold", (chain) => chain.toggleBold()),
      command("I", "Italic", (chain) => chain.toggleItalic()),
      command("S", "Strikethrough", (chain) => chain.toggleStrike()),
      command("H", "Heading", (chain) => chain.toggleHeading({ level: 2 })),
      command("•", "Bullet list", (chain) => chain.toggleBulletList()),
      command("1.", "Ordered list", (chain) => chain.toggleOrderedList()),
      command("❝", "Block quote", (chain) => chain.toggleBlockquote()),
      command("</>", "Code block", (chain) => chain.toggleCodeBlock()),
      command("↶", "Undo", (chain) => chain.undo()),
      command("↷", "Redo", (chain) => chain.redo()),
    );
    return controls;
  };

  const initializeWithin = (root = document) => {
    const tiptap = window.CuebookTiptap;
    if (!tiptap?.Editor || !tiptap.StarterKit || !tiptap.Markdown) return;

    for (const textarea of textareasWithin(root)) {
      if (editors.has(textarea)) continue;

      const container = document.createElement("div");
      container.className = "markdown-editor";
      const content = document.createElement("div");
      content.className = "markdown-editor-content";
      container.append(content);
      textarea.after(container);
      textarea.hidden = true;

      const editor = new tiptap.Editor({
        element: content,
        extensions: [
          tiptap.StarterKit.configure({
            link: { openOnClick: false },
          }),
          tiptap.Markdown,
        ],
        content: textarea.value,
        contentType: "markdown",
        onUpdate: ({ editor: updatedEditor }) => {
          textarea.value = updatedEditor.getMarkdown();
        },
      });
      container.prepend(toolbar(editor));
      editors.set(textarea, { container, editor });
    }
  };

  const destroyWithin = (root) => {
    for (const textarea of textareasWithin(root)) {
      const instance = editors.get(textarea);
      if (!instance) continue;
      textarea.value = instance.editor.getMarkdown();
      instance.editor.destroy();
      instance.container.remove();
      textarea.hidden = false;
      editors.delete(textarea);
    }
  };

  const synchronizeWithin = (root) => {
    for (const textarea of textareasWithin(root)) {
      const instance = editors.get(textarea);
      if (instance) textarea.value = instance.editor.getMarkdown();
    }
  };

  document.addEventListener("submit", (event) => synchronizeWithin(event.target), true);
  document.addEventListener("htmx:afterSwap", (event) => initializeWithin(event.detail?.target));
  document.addEventListener("htmx:beforeCleanupElement", (event) => destroyWithin(event.detail?.elt));

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", () => initializeWithin(document), { once: true });
  } else {
    initializeWithin(document);
  }

  window.CuebookMarkdownEditor = Object.freeze({ initializeWithin, destroyWithin, synchronizeWithin });
})();
