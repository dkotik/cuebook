const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

function element(tagName = "div", markdown = false) {
  const node = {
    tagName: tagName.toUpperCase(),
    dataset: markdown ? { markdownEditor: "" } : {},
    value: "",
    children: [],
    matches(selector) {
      return selector === "textarea[data-markdown-editor]" && this.tagName === "TEXTAREA" && Object.hasOwn(this.dataset, "markdownEditor");
    },
    append(...children) {
      for (const child of children) {
        child.parentElement = this;
        this.children.push(child);
      }
    },
    querySelectorAll(selector) {
      assert.equal(selector, "textarea[data-markdown-editor]");
      const found = [];
      const visit = (parent) => {
        for (const child of parent.children) {
          if (child.matches(selector)) found.push(child);
          visit(child);
        }
      };
      visit(this);
      return found;
    },
  };
  return node;
}

function browser(initialRoot = element("main")) {
  const listeners = new Map();
  const instances = [];
  const document = {
    readyState: "complete",
    addEventListener(name, listener, options) {
      if (!listeners.has(name)) listeners.set(name, []);
      listeners.get(name).push({ listener, options });
    },
    querySelectorAll: (...args) => initialRoot.querySelectorAll(...args),
  };
  class EasyMDE {
    constructor(options) {
      this.options = options;
      this.textarea = options.element;
      this.current = this.textarea.value;
      this.destroyed = false;
      this.changeListeners = [];
      this.focuses = 0;
      this.codemirror = {
        on: (name, listener) => {
          assert.equal(name, "change");
          this.changeListeners.push(listener);
        },
        focus: () => { this.focuses++; },
      };
      instances.push(this);
    }
    value() { return this.current; }
    setValue(value) {
      this.current = value;
      for (const listener of this.changeListeners) listener();
    }
    toTextArea() {
      this.textarea.value = this.current;
      this.destroyed = true;
    }
  }
  const window = {
    EasyMDE,
    DOMPurify: { sanitize: (html) => `sanitized:${html}` },
  };
  vm.runInContext(fs.readFileSync(path.join(__dirname, "markdown-editor.js"), "utf8"), vm.createContext({ document, window }));

  return {
    document,
    instances,
    dispatch(name, event = {}) {
      for (const { listener } of listeners.get(name) || []) listener(event);
    },
  };
}

test("initializes marked textareas once and sanitizes EasyMDE previews", () => {
  const root = element("main");
  const textarea = element("textarea", true);
  textarea.value = "# Existing Markdown";
  root.append(textarea);
  const app = browser(root);

  assert.equal(app.instances.length, 1);
  assert.equal(app.instances[0].options.element, textarea);
  assert.equal(app.instances[0].options.renderingConfig.sanitizerFunction("<script>unsafe</script>"), "sanitized:<script>unsafe</script>");
  app.dispatch("htmx:afterSwap", { detail: { target: root } });
  assert.equal(app.instances.length, 1);
});

test("initializes dynamically swapped multiline fields", () => {
  const app = browser();
  const swapped = element("section");
  const textarea = element("textarea", true);
  swapped.append(textarea);

  app.dispatch("htmx:afterSwap", { detail: { target: swapped } });

  assert.equal(app.instances.length, 1);
  assert.equal(app.instances[0].options.element, textarea);
});

test("keeps the source textarea synchronized for copy and form submission", () => {
  const root = element("form");
  const textarea = element("textarea", true);
  root.append(textarea);
  const app = browser(root);
  const editor = app.instances[0];

  editor.setValue("## Edited\nMarkdown");
  assert.equal(textarea.value, "## Edited\nMarkdown");
  editor.setValue("latest value");
  textarea.value = "stale value";
  app.dispatch("submit", { target: root });
  assert.equal(textarea.value, "latest value");
});

test("destroys editors before HTMX removes their fields and preserves content", () => {
  const root = element("section");
  const textarea = element("textarea", true);
  root.append(textarea);
  const app = browser(root);
  const editor = app.instances[0];
  editor.setValue("keep this Markdown");

  app.dispatch("htmx:beforeCleanupElement", { detail: { elt: root } });

  assert.equal(editor.destroyed, true);
  assert.equal(textarea.value, "keep this Markdown");
});
