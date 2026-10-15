const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

function browser({ move = false, missingInput = false, disabled = false, folded = false } = {}) {
  const handlers = new Map();
  class Element {
    constructor(selectors = []) {
      this.selectors = new Set(selectors);
      this.dataset = {};
      this.parentElement = null;
    }
    closest(selector) {
      return this.selectors.has(selector) ? this : this.parentElement?.closest(selector) || null;
    }
    matches(selector) { return this.selectors.has(selector); }
  }
  class HTMLFormElement extends Element {}
  class HTMLDialogElement extends Element {
    constructor() {
      super(["dialog", "dialog[data-file-picker-dialog]"]);
      this.open = false;
    }
    querySelectorAll(selector) {
      return selector.includes(".file-picker-option") ? options : [...options, unrelated];
    }
    querySelector(selector) {
      if (selector === "[data-file-picker-confirm]") return confirm;
      return this.querySelectorAll(selector).find((option) => option.checked && !option.disabled) || null;
    }
    showModal() { this.open = true; }
    close() { this.open = false; }
  }
  const dialog = new HTMLDialogElement();
  if (!move) dialog.dataset = { valueInput: "file", valueLabel: "label" };
  const options = ["source.cue", "nested/target.cue"].map((value) => {
    const option = new Element(['input[type="radio"]', '.file-picker-option input[type="radio"]']);
    option.value = value;
    option.checked = false;
    option.disabled = disabled && value === "source.cue";
    option.parentElement = dialog;
    return option;
  });
  const unrelated = new Element(['input[type="radio"]']);
  unrelated.value = "unrelated";
  unrelated.checked = true;
  unrelated.parentElement = dialog;
  const input = { value: "source.cue" };
  const label = { textContent: "source.cue" };
  const opener = new Element(["[data-file-picker-open]"]);
  opener.dataset.filePickerOpen = "picker";
  const confirm = new Element(["[data-file-picker-confirm]"]);
  confirm.disabled = true;
  confirm.parentElement = dialog;
  const cancel = new Element(["[data-file-picker-dismiss]"]);
  cancel.parentElement = dialog;
  const form = new HTMLFormElement();
  form.parentElement = dialog;
  const children = new Element([".tree-children"]);
  children.hidden = folded;
  const toggle = { expanded: "false", setAttribute(name, value) {
    assert.equal(name, "aria-expanded");
    this.expanded = value;
  } };
  const folder = new Element();
  folder.parentElement = dialog;
  folder.querySelector = () => toggle;
  children.parentElement = folder;
  if (folded) options[0].parentElement = children;
  const elements = new Map([["picker", dialog], ["label", label]]);
  if (!missingInput) elements.set("file", input);
  const document = {
    getElementById: (id) => elements.get(id) || null,
    addEventListener(name, listener) {
      if (!handlers.has(name)) handlers.set(name, []);
      handlers.get(name).push(listener);
    },
  };
  const context = vm.createContext({ document, Element, HTMLDialogElement, HTMLFormElement });
  vm.runInContext(fs.readFileSync(path.join(__dirname, "file-picker.js"), "utf8"), context);
  function dispatch(name, target) {
    for (const listener of handlers.get(name) || []) listener({ target });
  }
  return {
    dialog, input, label, options, unrelated, confirm, children, toggle,
    open: () => dispatch("click", opener),
    choose(index) {
      for (const option of options) option.checked = false;
      options[index].checked = true;
      dispatch("change", options[index]);
    },
    cancel: () => dispatch("click", cancel),
    commit: () => dispatch("click", confirm),
    submit: () => dispatch("submit", form),
  };
}

test("file picker opening restores only its own selection", async (t) => {
  const cases = [
    { name: "add starts at the committed file", expected: 0 },
    { name: "move starts without a destination", move: true, expected: -1 },
    { name: "a disabled current file cannot be selected", disabled: true, expected: -1 },
    { name: "a missing target cannot retain stale choices", missingInput: true, expected: -1 },
    { name: "the selected file is revealed in folded folders", folded: true, expected: 0 },
  ];
  for (const scenario of cases) {
    await t.test(scenario.name, () => {
      const app = browser(scenario);
      app.options[1].checked = true;
      app.open();
      assert.equal(app.dialog.open, true);
      assert.equal(app.options.findIndex((option) => option.checked), scenario.expected);
      assert.equal(app.confirm.disabled, scenario.expected < 0);
      assert.equal(app.unrelated.checked, true, "unrelated radios must not be reset");
      if (scenario.folded) {
        assert.equal(app.children.hidden, false);
        assert.equal(app.toggle.expanded, "true");
      }
    });
  }
});

test("file picker selection and confirmation", async (t) => {
  const cases = [
    { name: "add keeps the changed radio until confirmation" },
    { name: "cancel leaves the original form untouched", cancel: true },
    { name: "move enables its submit button without changing the add form", move: true },
  ];
  for (const scenario of cases) {
    await t.test(scenario.name, () => {
      const app = browser(scenario);
      app.open();
      app.choose(1);
      assert.equal(app.options[1].checked, true, "change must not restore the initial selection");
      assert.equal(app.confirm.disabled, false);
      assert.equal(app.input.value, "source.cue", "selection is not committed until confirmed");
      assert.equal(app.unrelated.checked, true);
      if (scenario.cancel) {
        app.cancel();
        assert.equal(app.dialog.open, false);
        assert.equal(app.input.value, "source.cue");
        app.open();
        assert.equal(app.options[0].checked, true);
      } else if (scenario.move) {
        app.commit();
        assert.equal(app.dialog.open, true, "move confirmation must allow native form submission");
        app.submit();
        assert.equal(app.dialog.open, false);
        assert.equal(app.input.value, "source.cue");
        app.open();
        assert.equal(app.options.every((option) => !option.checked), true);
      } else {
        app.commit();
        assert.equal(app.dialog.open, false);
        assert.equal(app.input.value, "nested/target.cue");
        assert.equal(app.label.textContent, "nested/target.cue");
        app.open();
        assert.equal(app.options[1].checked, true);
      }
    });
  }
});
