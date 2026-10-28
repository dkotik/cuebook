const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

function browser({ canArchive = true } = {}) {
  const handlers = new Map();
  const requests = [];
  const dispatched = [];
  const classes = () => {
    const names = new Set();
    return {
      add: (...values) => values.forEach((value) => names.add(value)),
      remove: (...values) => values.forEach((value) => names.delete(value)),
      contains: (value) => names.has(value),
    };
  };
  class Element {
    constructor(selectors = []) {
      this.selectors = new Set(selectors);
      this.dataset = {};
      this.classList = classes();
      this.parentElement = null;
    }
    closest(selector) {
      return this.selectors.has(selector) ? this : this.parentElement?.closest(selector) || null;
    }
    getBoundingClientRect() { return { width: 100, height: 80 }; }
  }
  class CustomEvent {
    constructor(type, options = {}) {
      this.type = type;
      this.detail = options.detail;
      this.cancelable = Boolean(options.cancelable);
      this.defaultPrevented = false;
    }
    preventDefault() { if (this.cancelable) this.defaultPrevented = true; }
  }

  const body = new Element();
  body.dataset.routePrefix = "/books";
  const workspace = {
    attributes: {},
    setAttribute(name, value) { this.attributes[name] = value; },
    removeAttribute(name) { delete this.attributes[name]; },
  };
  const archiveTarget = new Element(["[data-archive-drop-target]"]);
  const card = new Element([".entry[data-entry-index][data-file]"]);
  card.dataset = { file: "source.cue", entryIndex: "2", canArchive: String(canArchive) };
  const handle = new Element(["[data-entry-drag-handle]"]);
  handle.parentElement = card;
  const document = {
    body,
    addEventListener(name, listener) {
      if (!handlers.has(name)) handlers.set(name, []);
      handlers.get(name).push(listener);
    },
    dispatchEvent(event) {
      dispatched.push(event);
      for (const listener of handlers.get(event.type) || []) listener(event);
      return !event.defaultPrevented;
    },
    getElementById: (id) => id === "workspace" ? workspace : null,
    querySelectorAll(selector) {
      if (selector.includes("[data-archive-drop-target].is-drop-target")) {
        return archiveTarget.classList.contains("is-drop-target") ? [archiveTarget] : [];
      }
      return [];
    },
  };
  const window = {
    htmx: {
      async ajax(method, url, options) {
        requests.push({ method, url, options });
      },
    },
    location: { reload() { throw new Error("unexpected reload"); } },
  };
  const context = vm.createContext({ document, Element, CustomEvent, window });
  vm.runInContext(fs.readFileSync(path.join(__dirname, "entry-move.js"), "utf8"), context);

  function dispatch(name, event) {
    event.target ||= handle;
    return Promise.all((handlers.get(name) || []).map((listener) => listener(event)));
  }
  const dataTransfer = {
    setData() {},
    setDragImage() {},
  };
  return {
    archiveTarget,
    card,
    dataTransfer,
    dispatched,
    handle,
    requests,
    workspace,
    dragStart: () => dispatch("dragstart", { target: handle, dataTransfer, preventDefault() {} }),
    dragOver: () => {
      let prevented = false;
      return dispatch("dragover", {
        target: archiveTarget,
        dataTransfer,
        preventDefault() { prevented = true; },
      }).then(() => prevented);
    },
    drop: async () => {
      let prevented = false;
      await dispatch("drop", { target: archiveTarget, preventDefault() { prevented = true; } });
      return prevented;
    },
  };
}

test("archive drop requests confirmation before posting", async () => {
  const app = browser();
  await app.dragStart();

  assert.equal(await app.dragOver(), true);
  assert.equal(app.archiveTarget.classList.contains("is-drop-target"), true);
  assert.equal(await app.drop(), true);
  assert.equal(app.archiveTarget.classList.contains("is-drop-target"), false);
  assert.equal(app.requests.length, 0, "dropping must not archive before confirmation");
  assert.equal(app.dispatched.length, 1);

  const request = app.dispatched[0];
  assert.equal(request.type, "cuebook:archive-confirmation-request");
  assert.equal(request.detail.sourceCard, app.card);
  assert.equal(request.detail.archiveTarget, app.archiveTarget);
  assert.equal(typeof request.detail.confirm, "function");
  await request.detail.confirm();

  assert.equal(app.requests.length, 1);
  assert.equal(app.requests[0].method, "POST");
  assert.equal(app.requests[0].url, "/books/archive");
  assert.equal(app.requests[0].options.target, "#workspace");
  assert.equal(app.requests[0].options.swap, "outerHTML");
  assert.deepEqual({ ...app.requests[0].options.values }, { file: "source.cue", entry: "2" });
  assert.equal(app.workspace.attributes["aria-busy"], undefined);
});

test("entries that cannot be archived are not accepted by the archive drop target", async () => {
  const app = browser({ canArchive: false });
  await app.dragStart();

  assert.equal(await app.dragOver(), false);
  assert.equal(app.archiveTarget.classList.contains("is-drop-target"), false);
  await app.drop();

  assert.equal(app.dispatched.length, 0);
  assert.equal(app.requests.length, 0);
});
