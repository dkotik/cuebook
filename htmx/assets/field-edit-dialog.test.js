const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}

function browser(options = {}) {
  const handlers = new Map();
  const writes = [];
  const flashes = [];
  const removals = [];
  const prefix = options.prefix || "";
  const key = "cuebook-cut:" + (prefix || "/");
  const stored = "pending cut";
  const otherKey = "cuebook-cut:/other";
  const storage = new Map([[key, stored], [otherKey, "unrelated cut"]]);
  const classes = (...initial) => {
    const names = new Set(initial);
    return {
      add: (name) => names.add(name),
      remove: (name) => names.delete(name),
      contains: (name) => names.has(name),
    };
  };
  class Element {
    constructor(matches = {}) {
      this.matches = matches;
      this.isConnected = true;
      this.isContentEditable = false;
      this.disabled = false;
      this.dataset = {};
    }
    closest(selector) { return this.matches[selector] || null; }
    hasAttribute(name) { return name === "data-field-edit-trigger" && this === opener; }
    focus() { document.activeElement = this; }
  }
  const body = new Element();
  body.dataset.routePrefix = prefix;
  const opener = new Element();
  let openerFocuses = 0;
  opener.focus = () => { openerFocuses++; document.activeElement = opener; };
  const modal = new Element();
  modal.id = "field-edit-modal";
  modal.classList = classes();
  modal.attributes = {};
  modal.setAttribute = (name, value) => { modal.attributes[name] = value; };
  modal.innerHTML = "dialog content";
  let form;
  let value;
  let copy;
  function swapForm() {
    form = new Element();
    value = new Element();
    value.type = options.type || "text";
    value.value = "original rendered value";
    value.defaultValue = value.value;
    form.elements = { namedItem: (name) => name === "value" ? value : null };
    copy = new Element();
    copy.matches = { "[data-field-edit-copy]": copy, "#field-edit-modal": modal, "form": form };
    form.matches = { "#field-edit-modal form": form, "#field-edit-modal": modal };
    modal.innerHTML = "dialog content";
    modal.querySelector = (selector) => {
      if (selector === "form") return form;
      if (selector === "[data-field-edit-copy]") return copy;
      return null;
    };
    modal.querySelectorAll = () => [value, copy];
    modal.contains = (element) => [form, value, copy].includes(element);
  }
  swapForm();
  const document = {
    body,
    activeElement: body,
    documentElement: { classList: classes() },
    addEventListener(name, listener) {
      if (!handlers.has(name)) handlers.set(name, new Set());
      handlers.get(name).add(listener);
    },
    getElementById: (id) => id === modal.id ? modal : null,
  };
  function dispatch(name, event) {
    const pending = [];
    for (const listener of handlers.get(name) || []) pending.push(listener(event));
    return Promise.all(pending);
  }
  const localStorage = {
    removeItem(name) {
      removals.push(name);
      if (options.storageFailure) throw new Error("storage blocked");
      storage.delete(name);
    },
  };
  const navigator = { clipboard: {
    writeText(text) {
      assert.equal(storage.has(key), false, "pending cut must be cleared before clipboard write");
      assert.equal(copy.disabled, true, "copy button must be disabled before clipboard write");
      writes.push(text);
      if (options.clipboardThrows) throw new Error("clipboard blocked");
      if (options.clipboardFailure) return Promise.reject(new Error("permission denied"));
      return options.pending ? options.pending.promise : Promise.resolve();
    },
  } };
  if (options.noClipboard) delete navigator.clipboard;
  if (options.noWriteText) navigator.clipboard = {};
  const htmx = {
    trigger(target, name, detail) {
      assert.equal(name, "cuebook:flash");
      assert.equal(typeof detail.message, "string");
      assert.ok(detail.message.length > 0);
      flashes.push(detail.message);
    },
  };
  const context = vm.createContext({ Element, document, localStorage, navigator, window: { htmx } });
  vm.runInContext(fs.readFileSync(path.join(__dirname, "field-edit-dialog.js"), "utf8"), context);
  dispatch("htmx:beforeRequest", { detail: { elt: opener } });
  if (!options.inactive) dispatch("htmx:afterSwap", { detail: { target: modal } });
  function event(target, overrides = {}) {
    return {
      target, defaultPrevented: false, repeat: false, isComposing: false,
      ctrlKey: false, metaKey: false, altKey: false, shiftKey: false,
      preventDefault() { this.defaultPrevented = true; },
      ...overrides,
    };
  }
  return {
    document, modal, opener, storage, key, stored, otherKey, writes, flashes, removals,
    get form() { return form; },
    get value() { return value; },
    get copy() { return copy; },
    get openerFocuses() { return openerFocuses; },
    swapForm,
    focus(kind) {
      const focused = new Element();
      if (kind === "contenteditable") focused.isContentEditable = true;
      else if (kind !== "non-control") focused.matches["input, textarea, select, button"] = focused;
      document.activeElement = focused;
    },
    click(target = copy) {
      const click = event(target);
      return { event: click, done: dispatch("click", click) };
    },
    nestedTarget() { return new Element({ "[data-field-edit-copy]": copy, "#field-edit-modal": modal }); },
    keydown(overrides = {}) {
      const keydown = event(document.activeElement, { key: "c", ctrlKey: true, ...overrides });
      return { event: keydown, done: dispatch("keydown", keydown) };
    },
  };
}

async function settle(action) {
  await action.done;
  await new Promise((resolve) => setImmediate(resolve));
}

function assertOpen(app) {
  assert.equal(app.modal.classList.contains("is-active"), true);
  assert.equal(app.modal.attributes["aria-hidden"], "false");
  assert.equal(app.document.documentElement.classList.contains("is-clipped"), true);
  assert.equal(app.modal.innerHTML, "dialog content");
  assert.equal(app.openerFocuses, 0);
}

function assertClosed(app) {
  assert.equal(app.modal.classList.contains("is-active"), false);
  assert.equal(app.modal.attributes["aria-hidden"], "true");
  assert.equal(app.document.documentElement.classList.contains("is-clipped"), false);
  assert.equal(app.modal.innerHTML, "");
  assert.equal(app.document.activeElement, app.opener);
  assert.equal(app.openerFocuses, 1);
}

test("field edit copy uses the live form value and existing dialog close behavior", async (t) => {
  const cases = [
    { name: "plain text", value: "edited text" },
    { name: "unicode", value: "Café 日本語 🎵" },
    { name: "multiline", type: "textarea", value: "first\nsecond\r\nthird" },
    { name: "empty", value: "" },
    { name: "boolean hidden value", type: "hidden", value: "false" },
    { name: "password current value", type: "password", value: "new secret, not the rendered value" },
    { name: "mounted app clears only its own cut", prefix: "/books", value: "mounted value" },
    { name: "delegated click from nested button content", nested: true, value: "nested click" },
  ];
  for (const scenario of cases) {
    await t.test(scenario.name, async () => {
      const app = browser(scenario);
      app.value.value = scenario.value;
      await settle(app.click(scenario.nested ? app.nestedTarget() : app.copy));
      assert.deepEqual(app.writes, [scenario.value]);
      assert.deepEqual(app.removals, [app.key]);
      assert.equal(app.storage.get(app.otherKey), "unrelated cut");
      assert.equal(app.flashes.length, 0);
      assertClosed(app);
    });
  }
});

test("field edit copy shortcuts accept Ctrl or Meta and preserve guarded defaults", async (t) => {
  const cases = [
    { name: "Ctrl+c", accepted: true },
    { name: "Ctrl+C", key: "C", accepted: true },
    { name: "Meta+c", ctrlKey: false, metaKey: true, accepted: true },
    { name: "Meta+C", ctrlKey: false, metaKey: true, key: "C", accepted: true },
    { name: "non-control focused", focus: "non-control", accepted: true },
    { name: "already prevented", defaultPrevented: true },
    { name: "repeat", repeat: true },
    { name: "composing", isComposing: true },
    { name: "Alt modifier", altKey: true },
    { name: "Shift modifier", shiftKey: true },
    { name: "no copy modifier", ctrlKey: false },
    { name: "different key", key: "x" },
    { name: "focused input", focus: "input" },
    { name: "focused textarea", focus: "textarea" },
    { name: "focused select", focus: "select" },
    { name: "focused button", focus: "button" },
    { name: "focused copy button", focusCopy: true },
    { name: "focused contenteditable", focus: "contenteditable" },
    { name: "inactive modal", inactive: true },
  ];
  for (const scenario of cases) {
    await t.test(scenario.name, async () => {
      const app = browser(scenario);
      if (scenario.focus) app.focus(scenario.focus);
      if (scenario.focusCopy) {
        app.copy.matches["input, textarea, select, button"] = app.copy;
        app.copy.focus();
      }
      app.value.value = "live shortcut value";
      const action = app.keydown(scenario);
      await settle(action);
      assert.equal(action.event.defaultPrevented, Boolean(scenario.accepted || scenario.defaultPrevented));
      assert.deepEqual(app.writes, scenario.accepted ? ["live shortcut value"] : []);
      assert.equal(app.flashes.length, 0);
      if (scenario.accepted) assertClosed(app);
      else {
        assert.deepEqual(app.removals, []);
        assert.equal(app.storage.get(app.key), app.stored);
        assert.equal(app.copy.disabled, false);
        if (!scenario.inactive) assertOpen(app);
      }
    });
  }
});

test("field edit copy failures flash and keep the dialog usable", async (t) => {
  const cases = [
    { name: "clipboard rejection", clipboardFailure: true, writes: 1 },
    { name: "synchronous clipboard exception", clipboardThrows: true, writes: 1 },
    { name: "clipboard unavailable", noClipboard: true, writes: 0 },
    { name: "writeText unavailable", noWriteText: true, writes: 0 },
    { name: "storage exception blocks copy", storageFailure: true, writes: 0 },
    { name: "shortcut clipboard rejection", shortcut: true, clipboardFailure: true, writes: 1 },
    { name: "shortcut storage exception", shortcut: true, storageFailure: true, writes: 0 },
  ];
  for (const scenario of cases) {
    await t.test(scenario.name, async () => {
      const app = browser(scenario);
      await settle(scenario.shortcut ? app.keydown() : app.click());
      assert.equal(app.writes.length, scenario.writes);
      assert.equal(app.flashes.length, 1);
      assert.equal(app.copy.disabled, false);
      if (scenario.storageFailure) assert.equal(app.storage.get(app.key), app.stored);
      else if (scenario.writes) assert.equal(app.storage.has(app.key), false);
      assertOpen(app);
    });
  }
});

test("inactive dialog and unrelated delegated clicks never copy", async (t) => {
  const cases = [
    { name: "inactive dialog copy button", inactive: true },
    { name: "unrelated click inside active dialog", unrelated: true },
  ];
  for (const scenario of cases) {
    await t.test(scenario.name, async () => {
      const app = browser(scenario);
      await settle(app.click(scenario.unrelated ? app.value : app.copy));
      assert.deepEqual(app.writes, []);
      assert.deepEqual(app.removals, []);
      assert.deepEqual(app.flashes, []);
      assert.equal(app.copy.disabled, false);
      if (!scenario.inactive) assertOpen(app);
    });
  }
});

test("pending clipboard copy closes only on success and never closes a swapped form", async (t) => {
  const cases = [
    { name: "successful copy waits and ignores duplicate clicks" },
    { name: "shortcut waits and ignores a second shortcut", shortcut: true },
    { name: "pending click ignores a shortcut", duplicateShortcut: true },
    { name: "rejected copy re-enables the button", failure: true },
    { name: "successful old copy leaves newly swapped dialog open", swap: true },
    { name: "failed old copy leaves newly swapped dialog open", swap: true, failure: true },
  ];
  for (const scenario of cases) {
    await t.test(scenario.name, async () => {
      const pending = deferred();
      const app = browser({ pending });
      app.value.value = "value when copy started";
      const action = scenario.shortcut ? app.keydown() : app.click();
      assert.deepEqual(app.writes, ["value when copy started"]);
      assert.equal(app.copy.disabled, true);
      assertOpen(app);
      const duplicate = scenario.shortcut || scenario.duplicateShortcut ? app.keydown() : app.click();
      assert.equal(app.writes.length, 1, "only one clipboard operation may be pending");
      app.value.value = "changed while pending";
      if (scenario.swap) app.swapForm();
      if (scenario.failure) pending.reject(new Error("permission denied"));
      else pending.resolve();
      await settle(action);
      await settle(duplicate);
      assert.equal(app.writes.length, 1);
      assert.deepEqual(app.removals, [app.key]);
      assert.equal(app.flashes.length, scenario.failure ? 1 : 0);
      if (scenario.failure || scenario.swap) {
        assertOpen(app);
        assert.equal(app.copy.disabled, false);
      } else assertClosed(app);
    });
  }
});
