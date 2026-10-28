const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

function browser({ value = "not_set", optional = "true", allowUnset = "false", name = "entry.0.value", label = "Enabled", withForm = true, missing = "", expanded = null } = {}) {
  class Element {
    constructor() {
      this.listeners = new Map();
      this.attributes = new Map();
      this.dataset = {};
      this.parentElement = null;
      this.classes = new Set();
      this.classList = {
        toggle: (name, enabled) => enabled ? this.classes.add(name) : this.classes.delete(name),
        contains: (name) => this.classes.has(name),
      };
    }
    addEventListener(name, listener) {
      if (!this.listeners.has(name)) this.listeners.set(name, new Set());
      this.listeners.get(name).add(listener);
    }
    removeEventListener(name, listener) {
      this.listeners.get(name)?.delete(listener);
    }
    dispatchEvent(event) {
      Object.defineProperty(event, "target", { configurable: true, value: this });
      let current = this;
      while (current) {
        for (const listener of current.listeners.get(event.type) || []) listener(event);
        current = event.bubbles ? current.parentElement : null;
      }
      return !event.defaultPrevented;
    }
    setAttribute(name, value) {
      const oldValue = this.attributes.get(name) ?? null;
      this.attributes.set(name, String(value));
      if (oldValue !== String(value)) this.attributeChangedCallback?.(name, oldValue, String(value));
    }
    removeAttribute(name) {
      const oldValue = this.attributes.get(name) ?? null;
      this.attributes.delete(name);
      if (oldValue !== null) this.attributeChangedCallback?.(name, oldValue, null);
    }
    hasAttribute(name) { return this.attributes.has(name); }
    getAttribute(name) { return this.attributes.get(name) ?? null; }
    querySelector(selector) {
      if (selector === 'input[type="hidden"]') return this.children.find((child) => child.type === "hidden") || null;
      if (selector === "button.boolean-toggle") return this.children.find((child) => child.classList.contains("boolean-toggle")) || null;
      throw new Error(`Unexpected selector: ${selector}`);
    }
    appendChild(child) {
      if (child.parentElement) child.parentElement.removeChild(child);
      this.children.push(child);
      child.parentElement = this;
      return child;
    }
    removeChild(child) {
      const index = this.children.indexOf(child);
      if (index >= 0) this.children.splice(index, 1);
      child.parentElement = null;
      return child;
    }
    cloneNode() {
      const clone = new Element();
      clone.attributes = new Map(this.attributes);
      clone.dataset = { ...this.dataset };
      clone.classes = new Set(this.classes);
      clone.type = this.type;
      clone.textContent = this.textContent;
      clone.title = this.title;
      return clone;
    }
    click() { this.dispatchEvent(new Event("click", { bubbles: true })); }
  }
  class HiddenInput extends Element {
    constructor() {
      super();
      this.type = "hidden";
      this.name = name;
      this.defaultValue = value;
    }
    get value() { return this.defaultValue; }
    set value(value) { this.defaultValue = String(value); }
  }
  class Form extends Element {
    constructor() {
      super();
      this.controls = [];
    }
    reset() {
      if (this.dispatchEvent(new Event("reset", { bubbles: true, cancelable: true }))) {
        for (const control of this.controls) control.value = control.defaultValue;
      }
    }
    serialize() {
      return new URLSearchParams(this.controls.map((control) => [control.name, control.value])).toString();
    }
  }
  const definitions = new Map();
  const context = vm.createContext({
    HTMLElement: Element,
    Event,
    queueMicrotask,
    customElements: { define: (name, element) => definitions.set(name, element) },
  });
  vm.runInContext(fs.readFileSync(path.join(__dirname, "boolean-toggle.js"), "utf8"), context);
  const toggle = new (definitions.get("boolean-toggle"))();
  toggle.dataset.optional = optional;
  toggle.dataset.allowUnset = allowUnset;
  if (expanded !== null) toggle.setAttribute("expanded", expanded);
  const input = new HiddenInput();
  const button = new Element();
  button.type = "button";
  button.classes = new Set(["button", "boolean-toggle", "is-link"]);
  button.setAttribute("aria-label", label);
  button.setAttribute("aria-pressed", "true");
  button.textContent = "stale";
  const form = withForm ? new Form() : null;
  input.form = form;
  if (form) form.controls.push(input);
  toggle.children = [input, button].filter((child) => child !== (missing === "input" ? input : missing === "button" ? button : null));
  for (const child of toggle.children) child.parentElement = toggle;
  function connect(nextForm = form) {
    toggle.parentElement = nextForm;
    input.form = nextForm;
    toggle.isConnected = true;
    toggle.connectedCallback();
  }
  function disconnect() {
    toggle.isConnected = false;
    toggle.disconnectedCallback();
    toggle.parentElement = null;
  }
  connect();
  return {
    toggle, input, button, form, connect, disconnect, Form,
    get buttons() { return toggle.children.filter((child) => child.classList.contains("boolean-toggle")); },
  };
}

function assertState(app, state) {
  if (app.toggle.hasAttribute("expanded")) {
    assertExpandedState(app, state);
    return;
  }
  const symbols = { true: "✔", false: "✖", not_set: "☐" };
  assert.equal(app.input.value, state);
  assert.equal(app.button.textContent, symbols[state]);
  assert.equal(app.button.classList.contains("is-link"), state === "true");
  assert.equal(app.button.classList.contains("has-text-info"), state === "true");
  assert.equal(app.button.classList.contains("has-text-danger"), state === "false");
  assert.equal(app.button.classList.contains("button"), true);
  assert.equal(app.button.classList.contains("boolean-toggle"), true);
  assert.equal(app.button.getAttribute("aria-pressed"), state === "not_set" ? "mixed" : state);
  assert.equal(app.button.title, `${app.button.getAttribute("aria-label")}: ${state}`);
  assert.equal(app.button.type, "button");
  assert.equal(app.toggle.children.length, 2);
  assert.equal(app.toggle.children[1], app.button, "the original button must be retained");
}

test("boolean toggle initializes and normalizes states", async (t) => {
  const cases = [
    { name: "optional true", value: "true", expected: "true" },
    { name: "optional false", value: "false", expected: "false" },
    { name: "optional unset", value: "not_set", expected: "not_set" },
    { name: "required true", optional: "false", value: "true", expected: "true" },
    { name: "required false", optional: "false", value: "false", expected: "false" },
    { name: "required unset remains unset initially", optional: "false", value: "not_set", expected: "not_set" },
    { name: "empty normalizes to unset", value: "", expected: "not_set" },
    { name: "unknown normalizes to unset", value: "unknown", expected: "not_set" },
    { name: "uppercase is unknown", value: "TRUE", expected: "not_set" },
    { name: "required unknown normalizes to unset", optional: "false", value: "unknown", expected: "not_set" },
    { name: "label is treated as text", value: "true", label: '<img src=x onerror="fail()">', expected: "true" },
  ];
  for (const scenario of cases) {
    await t.test(scenario.name, () => assertState(browser(scenario), scenario.expected));
  }
});

test("boolean toggle cycles through native button clicks", async (t) => {
  const cases = [
    { name: "optional full cycle", value: "not_set", states: ["true", "false", "not_set", "true"] },
    { name: "optional begins true", value: "true", states: ["false", "not_set", "true"] },
    { name: "required skips unset", optional: "false", value: "false", states: ["true", "false", "true", "false"] },
    { name: "required add allows all states", optional: "false", allowUnset: "true", value: "false", states: ["not_set", "true", "false", "not_set"] },
    { name: "optional add allows all states", optional: "true", allowUnset: "true", value: "not_set", states: ["true", "false", "not_set"] },
    { name: "required unset becomes true", optional: "false", value: "not_set", states: ["true", "false", "true"] },
    { name: "required unknown becomes true", optional: "false", value: "invalid", states: ["true", "false"] },
    { name: "no form is supported", withForm: false, states: ["true", "false", "not_set"] },
  ];
  for (const scenario of cases) {
    await t.test(scenario.name, () => {
      const app = browser(scenario);
      for (const state of scenario.states) {
        app.button.click();
        assertState(app, state);
      }
    });
  }
});

function assertExpandedState(app, expected) {
  assert.equal(app.input.value, expected);
  assert.equal(app.toggle.getAttribute("role"), "group");
  assert.equal(app.toggle.getAttribute("aria-label"), "Enabled");
  assert.deepEqual(app.buttons.map((button) => button.dataset.booleanState), ["true", "false"]);
  assert.equal(app.buttons.length, 2);
  assert.equal(app.buttons[0], app.button, "the original button must be retained");
  for (const button of app.buttons) {
    const buttonState = button.dataset.booleanState;
    const selected = buttonState === expected;
    const label = `Enabled: ${buttonState}`;
    assert.equal(button.textContent, buttonState === "true" ? "✔" : "✖");
    assert.equal(button.getAttribute("aria-label"), label);
    assert.equal(button.title, label);
    assert.equal(button.getAttribute("aria-pressed"), String(selected));
    assert.equal(button.classList.contains("is-link"), selected && buttonState === "true");
    assert.equal(button.classList.contains("has-text-info"), selected && buttonState === "true");
    assert.equal(button.classList.contains("has-text-danger"), selected && buttonState === "false");
    assert.equal(button.type, "button");
  }
}

test("expanded boolean toggle represents three states with two buttons", async (t) => {
  const cases = [
    { name: "unset by default", expanded: "true", expected: "not_set" },
    { name: "empty value defaults to unset", expanded: "true", value: "", expected: "not_set" },
    { name: "unknown value defaults to unset", expanded: "true", value: "invalid", expected: "not_set" },
    { name: "empty attribute expands true state", expanded: "", value: "true", expected: "true" },
    { name: "false-valued attribute still expands false state", expanded: "false", value: "false", expected: "false" },
    { name: "arbitrary attribute value expands unset state", expanded: "anything", value: "not_set", expected: "not_set" },
    { name: "required toggle can clear either value", expanded: "true", optional: "false", expected: "not_set" },
    { name: "required add toggle can clear either value", expanded: "true", optional: "false", allowUnset: "true", expected: "not_set" },
    { name: "without a form", expanded: "true", withForm: false, expected: "not_set" },
  ];
  for (const scenario of cases) {
    await t.test(scenario.name, async () => {
      const app = browser(scenario);
      assertExpandedState(app, scenario.expected);
      // Repeated clicks clear either value; clicking the other button switches directly.
      for (const state of ["true", "false"]) {
        const button = app.buttons.find((button) => button.dataset.booleanState === state);
        if (app.input.value === state) {
          button.click();
          assertExpandedState(app, "not_set");
        }
        button.click();
        assertExpandedState(app, state);
        button.click();
        assertExpandedState(app, "not_set");
        button.click();
        assertExpandedState(app, state);
      }
      app.form?.reset();
      await Promise.resolve();
      assertExpandedState(app, app.form ? scenario.expected : "false");
    });
  }
});

test("expanded boolean toggle responds to attribute changes", async () => {
  const app = browser();
  app.toggle.setAttribute("expanded", "false");
  assert.equal(app.buttons.length, 2);
  app.buttons.find((button) => button.dataset.booleanState === "false").click();
  assert.equal(app.input.value, "false");

  app.toggle.removeAttribute("expanded");
  assert.equal(app.buttons.length, 1);
  assert.equal(app.toggle.children.length, 2);
  assert.equal(app.toggle.getAttribute("role"), null);
  assert.equal(app.button.getAttribute("aria-label"), "Enabled");
  app.button.click();
  assertState(app, "not_set");

  app.toggle.setAttribute("expanded", "true");
  assertExpandedState(app, "not_set");
  const falseButton = app.buttons[1];
  app.connect();
  assert.equal(app.buttons[1], falseButton, "reconnecting must reuse the false button");
  falseButton.click();
  assertExpandedState(app, "false");
  falseButton.click();
  assertExpandedState(app, "not_set");
});

test("boolean toggle updates serialized hidden values and bubbles native events", async (t) => {
  const cases = [
    { name: "entry field", fieldName: "entry.12.value" },
    { name: "standalone value", fieldName: "value" },
    { name: "required add can submit unset for validation", fieldName: "entry.0.value", optional: "false", allowUnset: "true" },
    { name: "expanded add field", fieldName: "entry.12.value", expanded: "true" },
    { name: "expanded edit field", fieldName: "value", expanded: "true" },
    { name: "expanded required field", fieldName: "value", optional: "false", expanded: "true" },
  ];
  for (const scenario of cases) {
    await t.test(scenario.name, () => {
      const app = browser({ ...scenario, name: scenario.fieldName });
      const events = [];
      for (const type of ["input", "change"]) {
        app.form.addEventListener(type, (event) => {
          assert.ok(event instanceof Event);
          assert.equal(event.target, app.input);
          assert.equal(event.bubbles, true);
          assertState(app, app.input.value);
          events.push([event.type, app.input.value]);
        });
      }
      assert.equal(app.form.serialize(), `${scenario.fieldName}=not_set`);
      for (const state of ["true", "false", "not_set"]) {
        const button = scenario.expanded && state !== "true" ? app.buttons[1] : app.button;
        button.click();
        assert.equal(app.form.serialize(), `${scenario.fieldName}=${state}`);
      }
      assert.deepEqual(events, [
        ["input", "true"], ["change", "true"],
        ["input", "false"], ["change", "false"],
        ["input", "not_set"], ["change", "not_set"],
      ]);
    });
  }
});

test("boolean toggle resets to its initial hidden value after the reset event", async (t) => {
  const cases = [
    { name: "unset default", value: "not_set", expected: "not_set" },
    { name: "true default", value: "true", expected: "true" },
    { name: "false default", value: "false", expected: "false" },
    { name: "empty default is normalized", value: "", expected: "not_set" },
    { name: "unknown default is normalized", value: "invalid", expected: "not_set" },
    { name: "expanded unset default", expanded: "true", expected: "not_set" },
    { name: "expanded true default", expanded: "true", value: "true", expected: "true" },
    { name: "expanded false default", expanded: "true", value: "false", expected: "false" },
    { name: "expanded empty default", expanded: "true", value: "", expected: "not_set" },
    { name: "expanded canceled reset", expanded: "true", value: "true", canceled: true, expected: "not_set" },
    { name: "expanded reset after reconnect", expanded: "true", value: "true", reconnect: true, expected: "true" },
    { name: "required default", optional: "false", value: "false", expected: "false" },
    { name: "canceled reset preserves state", value: "false", canceled: true, expected: "not_set" },
    { name: "reset after reconnect preserves original default", value: "true", reconnect: true, expected: "true" },
    { name: "disconnect before microtask prevents stale update", value: "false", disconnect: true, expected: "not_set" },
  ];
  for (const scenario of cases) {
    await t.test(scenario.name, async () => {
      const app = browser(scenario);
      app.button.click();
      const changed = app.input.value;
      if (scenario.reconnect) {
        app.disconnect();
        app.connect();
      }
      const events = [];
      app.form.addEventListener("input", (event) => events.push(event.type));
      app.form.addEventListener("change", (event) => events.push(event.type));
      if (scenario.canceled) app.form.addEventListener("reset", (event) => event.preventDefault());
      app.form.reset();
      assertState(app, changed);
      if (scenario.disconnect) app.disconnect();
      await Promise.resolve();
      assertState(app, scenario.expected);
      assert.deepEqual(events, [], "reset must not emit input or change events");
    });
  }
});

test("boolean toggle cleans up listeners across reconnections", async (t) => {
  const cases = [
    { name: "same form" },
    { name: "different form", move: true },
    { name: "without a form", withForm: false },
  ];
  for (const scenario of cases) {
    await t.test(scenario.name, async () => {
      const app = browser(scenario);
      const events = [];
      app.input.addEventListener("change", () => events.push(app.input.value));
      app.disconnect();
      app.button.click();
      app.form?.reset();
      await Promise.resolve();
      assertState(app, "not_set");
      assert.deepEqual(events, []);
      const nextForm = scenario.move ? new app.Form() : app.form;
      for (let index = 0; index < 3; index++) {
        app.connect(nextForm);
        app.disconnect();
      }
      app.connect(nextForm);
      app.button.click();
      assertState(app, "true");
      assert.deepEqual(events, ["true"], "reconnection must not duplicate listeners");
      if (scenario.move) {
        app.form.reset();
        await Promise.resolve();
        assertState(app, "true");
      }
      if (nextForm) {
        nextForm.reset();
        await Promise.resolve();
        assertState(app, "not_set");
      }
    });
  }
});

test("boolean toggle tolerates incomplete markup", async (t) => {
  const cases = [
    { name: "missing input", missing: "input" },
    { name: "missing button", missing: "button" },
  ];
  for (const scenario of cases) {
    await t.test(scenario.name, () => {
      const app = browser(scenario);
      app.button.click();
      app.disconnect();
      app.connect();
      assert.equal(app.input.value, "not_set");
    });
  }
});
