const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

function browser(options = {}) {
  const handlers = new Map();
  const storage = new Map();
  const writes = [];
  const requests = [];
  const flashes = [];
  const prefix = options.prefix || "";
  const key = "cuebook-cut:" + (prefix || "/");
  const record = { file: "source.cue", entry: "1", fingerprint: "fingerprint" };
  const stored = JSON.stringify(record);
  if (options.pendingCut) storage.set(key, stored);
  const status = { textContent: "" };
  const source = { textContent: '{Name: "Café & friends"}\r\n' };
  class Element {
    constructor(matches = {}) { this.matches = matches; }
    closest(selector) { return this.matches[selector] || null; }
  }
  const actions = {
    querySelector: () => status,
    querySelectorAll: () => [copy, cut],
  };
  const copy = new Element();
  copy.disabled = false;
  copy.dataset = { entryCopy: "entry-cue-1" };
  copy.hasAttribute = () => false;
  copy.matches = { "[data-entry-copy], [data-entry-cut]": copy, ".entry-actions": actions };
  const cut = new Element();
  cut.disabled = false;
  cut.dataset = { entryCut: "entry-cue-1", file: record.file, entryIndex: record.entry, cutFingerprint: record.fingerprint };
  cut.hasAttribute = (name) => name === "data-entry-cut";
  cut.matches = { "[data-entry-copy], [data-entry-cut]": cut, ".entry-actions": actions };
  const workspace = {};
  const destination = { dataset: { pasteFile: "destination.cue", pasteUrl: prefix + "/paste" } };
  const document = {
    body: { dataset: { routePrefix: prefix } },
    addEventListener(name, listener) {
      if (!handlers.has(name)) handlers.set(name, new Set());
      handlers.get(name).add(listener);
    },
    removeEventListener(name, listener) { handlers.get(name)?.delete(listener); },
    getElementById: (id) => id === "workspace" ? workspace : source,
    querySelector: () => options.noDestination ? null : destination,
  };
  async function dispatch(name, event) {
    for (const listener of [...(handlers.get(name) || [])]) await listener(event);
  }
  const localStorage = {
    getItem(name) {
      if (options.storageFailure === "get") throw new Error("blocked");
      return storage.get(name) || null;
    },
    removeItem(name) {
      if (options.storageFailure === "remove") throw new Error("blocked");
      storage.delete(name);
    },
    setItem(name, value) {
      if (options.storageFailure === "set") throw new Error("blocked");
      storage.set(name, value);
    },
  };
  const navigator = { clipboard: {
    async writeText(text) {
      assert.equal(storage.has(key), false, "copy/cut must clear older cut state before writing");
      writes.push(text);
      if (options.clipboardFailure) throw new Error("permission denied");
    },
  } };
  const htmx = {
    trigger: (_, name, detail) => { assert.equal(name, "cuebook:flash"); flashes.push(detail.message); },
    async ajax(method, route, config) {
      requests.push({ method, route, ...config });
      if (options.requestFailure) throw new Error("network failure");
      if (options.duringRequest) options.duringRequest(storage, key);
      await dispatch("htmx:afterRequest", {
        detail: { requestConfig: { path: route, parameters: config.values }, xhr: { status: options.responseStatus || 200 } },
      });
    },
  };
  const context = vm.createContext({ Element, document, localStorage, navigator, window: { htmx } });
  for (const name of ["entry-copy.js", "entry-paste.js"]) {
    vm.runInContext(fs.readFileSync(path.join(__dirname, "assets", name), "utf8"), context);
  }
  return {
    key, stored, record, storage, source, writes, requests, flashes, status, copy, cut, handlers,
    click: (button) => dispatch("click", { target: button }),
    paste: (text = source.textContent, editable = false) => {
      const target = new Element();
      if (editable) target.matches["input, textarea, select"] = target;
      return dispatch("paste", {
        target,
        clipboardData: { getData: (type) => { assert.equal(type, "text/plain"); return text; } },
        preventDefault() {},
      });
    },
  };
}

test("copy and cut clipboard state", async (t) => {
  const cases = [
    { name: "copy always clears pending cut", pendingCut: true },
    { name: "cut stores source identity only after clipboard success", cut: true },
    { name: "cut replaces an older cut", cut: true, pendingCut: true },
    { name: "failed copy still clears cut", pendingCut: true, clipboardFailure: true },
    { name: "failed cut does not retain an older cut", cut: true, pendingCut: true, clipboardFailure: true },
    { name: "storage removal failure blocks clipboard changes", pendingCut: true, storageFailure: "remove" },
    { name: "storage write failure leaves clipboard as an ordinary copy", cut: true, storageFailure: "set" },
    { name: "cut state is scoped to the mounted app", cut: true, prefix: "/books" },
  ];
  for (const scenario of cases) {
    await t.test(scenario.name, async () => {
      const app = browser(scenario);
      await app.click(scenario.cut ? app.cut : app.copy);
      if (scenario.storageFailure === "remove") {
        assert.equal(app.writes.length, 0);
        assert.equal(app.storage.get(app.key), app.stored);
      } else {
        assert.equal(app.writes[0], app.source.textContent);
        if (scenario.cut && !scenario.clipboardFailure && !scenario.storageFailure) {
          assert.deepEqual(JSON.parse(app.storage.get(app.key)), app.record);
          assert.ok(!app.storage.get(app.key).includes(app.source.textContent));
          assert.match(app.status.textContent, /Cut/);
        } else {
          assert.equal(app.storage.has(app.key), false);
        }
      }
      if (scenario.clipboardFailure || scenario.storageFailure) assert.equal(app.flashes.length, 1);
      assert.equal(app.requests.length, 0, "cut must not move before paste");
      assert.equal(app.copy.disabled, false);
      assert.equal(app.cut.disabled, false);
    });
  }
});

test("paste forwards raw source and consumes cut only on success", async (t) => {
  const cases = [
    { name: "plain copy paste forwards invalid source unchanged", text: "not CUE at all" },
    { name: "empty clipboard reaches server unchanged", text: "" },
    { name: "cut paste forwards source and metadata", pendingCut: true },
    { name: "cut errors retain pending state", pendingCut: true, responseStatus: 204 },
    { name: "server error retains pending state", pendingCut: true, responseStatus: 500 },
    { name: "network failure retains pending state", pendingCut: true, requestFailure: true },
    { name: "new cut during request is not cleared", pendingCut: true, duringRequest: (storage, key) => storage.set(key, "newer cut") },
    { name: "mounted cut paste uses scoped state", prefix: "/books", pendingCut: true },
    { name: "editable controls paste normally", editable: true, pendingCut: true },
    { name: "non-list view does not paste entries", noDestination: true, pendingCut: true },
    { name: "unreadable storage does not risk an accidental move", storageFailure: "get", pendingCut: true },
  ];
  for (const scenario of cases) {
    await t.test(scenario.name, async () => {
      const app = browser(scenario);
      const text = scenario.text === undefined ? app.source.textContent : scenario.text;
      await app.paste(text, scenario.editable);
      const blocked = scenario.editable || scenario.noDestination || scenario.storageFailure;
      assert.equal(app.requests.length, blocked ? 0 : 1);
      if (!blocked) {
        const request = app.requests[0];
        assert.equal(request.method, "POST");
        assert.equal(request.route, (scenario.prefix || "") + "/paste");
        assert.equal(request.values.source, text);
        assert.equal(request.values.file, "destination.cue");
        if (scenario.pendingCut) {
          assert.equal(request.values.cut_file, app.record.file);
          assert.equal(request.values.cut_entry, app.record.entry);
          assert.equal(request.values.cut_fingerprint, app.record.fingerprint);
        } else assert.equal(request.values.cut_file, undefined);
      }
      if (scenario.duringRequest) assert.equal(app.storage.get(app.key), "newer cut");
      else if (scenario.pendingCut) {
        const succeeds = !blocked && !scenario.requestFailure && (!scenario.responseStatus || scenario.responseStatus === 200);
        assert.equal(app.storage.has(app.key), !succeeds);
      }
      assert.equal(app.handlers.get("htmx:afterRequest")?.size || 0, 0, "request listeners must be cleaned up");
    });
  }
});

test("cut then copy then paste cannot move", async () => {
  const app = browser();
  await app.click(app.cut);
  assert.equal(app.storage.has(app.key), true);
  await app.click(app.copy);
  assert.equal(app.storage.has(app.key), false);
  await app.paste();
  assert.equal(app.requests[0].values.cut_file, undefined);
});
