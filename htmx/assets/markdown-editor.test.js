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
    hidden: false,
    className: "",
    children: [],
    attributes: new Map(),
    listeners: new Map(),
    matches(selector) {
      return selector === "textarea[data-markdown-editor]" && this.tagName === "TEXTAREA" && Object.hasOwn(this.dataset, "markdownEditor");
    },
    setAttribute(name, value) { this.attributes.set(name, value); },
    addEventListener(name, listener) { this.listeners.set(name, listener); },
    append(...children) {
      for (const child of children) {
        child.parentElement = this;
        this.children.push(child);
      }
    },
    prepend(child) {
      child.parentElement = this;
      this.children.unshift(child);
    },
    after(sibling) {
      const siblings = this.parentElement.children;
      siblings.splice(siblings.indexOf(this) + 1, 0, sibling);
      sibling.parentElement = this.parentElement;
    },
    remove() {
      const siblings = this.parentElement?.children;
      if (siblings) siblings.splice(siblings.indexOf(this), 1);
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
    createElement: (tagName) => element(tagName),
    addEventListener(name, listener, options) {
      if (!listeners.has(name)) listeners.set(name, []);
      listeners.get(name).push({ listener, options });
    },
    querySelectorAll: (...args) => initialRoot.querySelectorAll(...args),
  };
  class Editor {
    constructor(options) {
      this.options = options;
      this.current = options.content;
      this.destroyed = false;
      instances.push(this);
    }
    getMarkdown() { return this.current; }
    setMarkdown(value) {
      this.current = value;
      this.options.onUpdate({ editor: this });
    }
    destroy() { this.destroyed = true; }
    chain() {
      const chain = new Proxy({}, { get: () => () => chain });
      return chain;
    }
  }
  const window = {
    CuebookTiptap: {
      Editor,
      StarterKit: { configure: (options) => ({ starterKit: options }) },
      Markdown: { name: "markdown" },
      TaskList: { name: "taskList" },
      TaskItem: { configure: (options) => ({ taskItem: options }) },
    },
  };
  vm.runInContext(fs.readFileSync(path.join(__dirname, "markdown-editor.js"), "utf8"), vm.createContext({ document, window }));

  return {
    instances,
    dispatch(name, event = {}) {
      for (const { listener } of listeners.get(name) || []) listener(event);
    },
  };
}

test("initializes marked textareas once with Tiptap Markdown", () => {
  const root = element("main");
  const textarea = element("textarea", true);
  textarea.value = "# Existing Markdown";
  root.append(textarea);
  const app = browser(root);

  assert.equal(app.instances.length, 1);
  assert.equal(app.instances[0].options.element.className, "markdown-editor-content");
  assert.equal(app.instances[0].options.content, "# Existing Markdown");
  assert.equal(app.instances[0].options.contentType, "markdown");
  assert.equal(textarea.hidden, true);
  assert.equal(root.children[1].className, "markdown-editor");
  app.dispatch("htmx:afterSwap", { detail: { target: root } });
  assert.equal(app.instances.length, 1);
});

test("registers task list extensions so checkboxes survive editing", () => {
  const root = element("main");
  root.append(element("textarea", true));
  const app = browser(root);

  const { extensions } = app.instances[0].options;
  assert.ok(extensions.some((extension) => extension.name === "taskList"));
  assert.ok(extensions.some((extension) => extension.taskItem?.nested === true));
});

test("initializes dynamically swapped multiline fields", () => {
  const app = browser();
  const swapped = element("section");
  const textarea = element("textarea", true);
  swapped.append(textarea);

  app.dispatch("htmx:afterSwap", { detail: { target: swapped } });

  assert.equal(app.instances.length, 1);
  assert.equal(app.instances[0].options.element.className, "markdown-editor-content");
});

test("keeps the source textarea synchronized for copy and form submission", () => {
  const root = element("form");
  const textarea = element("textarea", true);
  root.append(textarea);
  const app = browser(root);
  const editor = app.instances[0];

  editor.setMarkdown("## Edited\nMarkdown");
  assert.equal(textarea.value, "## Edited\nMarkdown");
  editor.setMarkdown("latest value");
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
  editor.setMarkdown("keep this Markdown");

  app.dispatch("htmx:beforeCleanupElement", { detail: { elt: root } });

  assert.equal(editor.destroyed, true);
  assert.equal(textarea.value, "keep this Markdown");
  assert.equal(textarea.hidden, false);
  assert.equal(root.children.length, 1);
});
