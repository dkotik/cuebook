const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

function browser({ canArchive = true } = {}) {
  const documentHandlers = new Map();
  const modalHandlers = new Map();
  const elements = new Map();
  const classes = () => {
    const names = new Set();
    return {
      add: (name) => names.add(name),
      remove: (name) => names.delete(name),
      contains: (name) => names.has(name),
    };
  };
  class Element {
    constructor(selectors = []) {
      this.selectors = new Set(selectors);
      this.listeners = new Map();
      this.attributes = {};
      this.classList = classes();
      this.dataset = {};
      this.isConnected = true;
    }
    addEventListener(name, listener) {
      if (!this.listeners.has(name)) this.listeners.set(name, []);
      this.listeners.get(name).push(listener);
    }
    closest(selector) { return this.selectors.has(selector) ? this : null; }
    matches(selector) { return this.selectors.has(selector); }
    focus() { document.activeElement = this; }
    setAttribute(name, value) { this.attributes[name] = value; }
  }
  class HTMLElement extends Element {}
  class HTMLFormElement extends HTMLElement {}

  const entryLabel = new HTMLElement();
  const confirmButton = new HTMLElement(["[data-confirmation-confirm]"]);
  const cancelButton = new HTMLElement([".archive-confirmation-cancel"]);
  const modal = new HTMLElement();
  modal.querySelector = (selector) => ({
    "[data-confirmation-entry]": entryLabel,
    "[data-confirmation-confirm]": confirmButton,
    ".archive-confirmation-cancel": cancelButton,
  })[selector] || null;
  modal.addEventListener = (name, listener) => modalHandlers.set(name, listener);
  modal.querySelectorAll = () => [cancelButton, confirmButton];
  elements.set("archive-confirmation", modal);

  const document = {
    activeElement: null,
    documentElement: { classList: classes() },
    getElementById: (id) => elements.get(id) || null,
    addEventListener(name, listener) {
      if (!documentHandlers.has(name)) documentHandlers.set(name, []);
      documentHandlers.get(name).push(listener);
    },
  };
  const context = vm.createContext({ document, Element, HTMLElement, HTMLFormElement });
  vm.runInContext(fs.readFileSync(path.join(__dirname, "archive-confirm.js"), "utf8"), context);

  const title = new HTMLElement();
  title.textContent = "Entry title";
  const handle = new HTMLElement(["[data-entry-drag-handle]"]);
  const sourceCard = new HTMLElement();
  sourceCard.dataset.canArchive = String(canArchive);
  sourceCard.querySelector = (selector) => selector === ".card-header-title" ? title : handle;
  const archiveTarget = new HTMLElement(["[data-archive-drop-target]"]);
  const dismissTarget = new HTMLElement(["[data-confirmation-dismiss]"]);
  let confirmed = 0;

  function requestArchive() {
    const event = {
      detail: { sourceCard, archiveTarget, confirm: () => { confirmed++; } },
      defaultPrevented: false,
      preventDefault() { this.defaultPrevented = true; },
    };
    for (const listener of documentHandlers.get("cuebook:archive-confirmation-request") || []) listener(event);
    return event;
  }

  return {
    archiveTarget,
    cancelButton,
    confirmButton,
    document,
    entryLabel,
    handle,
    modal,
    modalHandlers,
    requestArchive,
    get confirmed() { return confirmed; },
    dismiss() { modalHandlers.get("click")({ target: dismissTarget }); },
    async confirm() {
      await confirmButton.listeners.get("click")[0]();
    },
  };
}

test("archive drop uses the existing confirmation dialog", async (t) => {
  const cases = [
    { name: "confirm invokes the archive action", confirm: true, expectedCalls: 1 },
    { name: "dismiss does not invoke the archive action", confirm: false, expectedCalls: 0 },
  ];
  for (const scenario of cases) {
    await t.test(scenario.name, async () => {
      const app = browser();
      const request = app.requestArchive();

      assert.equal(request.defaultPrevented, true, "the confirmation controller handles the request");
      assert.equal(app.modal.classList.contains("is-active"), true);
      assert.equal(app.modal.attributes["aria-hidden"], "false");
      assert.equal(app.entryLabel.textContent, "“Entry title”");
      assert.equal(app.document.documentElement.classList.contains("is-clipped"), true);
      if (scenario.confirm) await app.confirm();
      else app.dismiss();

      assert.equal(app.confirmed, scenario.expectedCalls);
      assert.equal(app.modal.classList.contains("is-active"), false);
      assert.equal(app.modal.attributes["aria-hidden"], "true");
      assert.equal(app.document.documentElement.classList.contains("is-clipped"), false);
      if (!scenario.confirm) assert.equal(app.document.activeElement, app.handle);
    });
  }
});

test("archive drop confirmation rejects entries that cannot be archived", () => {
  const app = browser({ canArchive: false });
  const request = app.requestArchive();

  assert.equal(request.defaultPrevented, false);
  assert.equal(app.modal.classList.contains("is-active"), false);
  assert.equal(app.confirmed, 0);
});
