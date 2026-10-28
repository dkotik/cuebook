const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

function browser(location) {
  const handlers = new Map();
  const window = { location: { href: location }, addEventListener: listen };
  function listen(name, listener) {
    handlers.set(name, listener);
  }
  const links = ["/book/", "/book/?folder=.archive", "/book/changes"].map((href) => ({
    href: new URL(href, location).href,
    current: false,
    setAttribute(name, value) {
      assert.equal(name, "aria-current");
      assert.equal(value, "page");
      this.current = true;
    },
    removeAttribute(name) {
      assert.equal(name, "aria-current");
      this.current = false;
    },
  }));
  const document = {
    addEventListener: listen,
    querySelectorAll(selector) {
      if (selector === "a.tree-file-link") return [];
      assert.equal(selector, ".file-nav > a");
      return links;
    },
  };
  const context = vm.createContext({
    document, window, URL, HTMLElement: class {},
    customElements: { get() { return true; } },
  });
  vm.runInContext(fs.readFileSync(path.join(__dirname, "file-tree.js"), "utf8"), context);
  return {
    selected: () => links.map((link) => link.current),
    navigate(href, event) {
      window.location.href = new URL(href, location).href;
      handlers.get(event)();
    },
  };
}

const cases = [
  { name: "Files", url: "/book/", selected: [true, false, false] },
  { name: "file", url: "/book/?file=notes.cue", selected: [false, false, false] },
  { name: "folder", url: "/book/?folder=nested", selected: [false, false, false] },
  { name: "Archive", url: "/book/?folder=.archive", selected: [false, true, false] },
  { name: "archived folder", url: "/book/?folder=.archive/nested", selected: [false, true, false] },
  { name: "archived file", url: "/book/?file=.archive/notes.cue", selected: [false, true, false] },
  { name: "archived entry", url: "/book/entry?file=.archive/notes.cue", selected: [false, true, false] },
  { name: "similar folder name", url: "/book/?folder=.archive-other", selected: [false, false, false] },
  { name: "Changes", url: "/book/changes", selected: [false, false, true] },
  { name: "search", url: "/book/search?q=notes", selected: [false, false, false] },
];

test("sidebar labels reflect the current page on load", async (t) => {
  for (const scenario of cases) {
    await t.test(scenario.name, () => {
      const page = browser(`https://example.test${scenario.url}`);
      assert.deepEqual(page.selected(), scenario.selected);
    });
  }
});

test("sidebar labels follow HTMX navigation and browser history", async (t) => {
  for (const event of ["htmx:afterSettle", "htmx:pushedIntoHistory", "popstate"]) {
    await t.test(event, () => {
      const page = browser("https://example.test/book/");
      for (const scenario of cases) {
        page.navigate(scenario.url, event);
        assert.deepEqual(page.selected(), scenario.selected, scenario.name);
      }
    });
  }
});
