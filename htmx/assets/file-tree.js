(() => {
  const storageKey = "cuebook-file-tree-folded";

  function readFoldedPaths() {
    try {
      const value = window.localStorage.getItem(storageKey);
      const paths = value ? JSON.parse(value) : [];
      return new Set(Array.isArray(paths) ? paths.filter((path) => typeof path === "string") : []);
    } catch (_) {
      return new Set();
    }
  }

  function writeFoldedPaths(paths) {
    try {
      window.localStorage.setItem(storageKey, JSON.stringify([...paths]));
    } catch (_) {
      // The tree remains interactive if browser storage is unavailable.
    }
  }

  class FileTreeNode extends HTMLElement {
    connectedCallback() {
      if (this.dataset.treeInitialized === "true") {
        return;
      }
      const toggle = this.querySelector(":scope > .tree-node-toggle");
      const children = this.querySelector(":scope > .tree-children");
      const nodePath = this.dataset.nodePath;
      if (!toggle || !children || !nodePath) {
        return;
      }

      this.dataset.treeInitialized = "true";
      this.setFolded(readFoldedPaths().has(nodePath), toggle, children);
      toggle.addEventListener("click", () => {
        const folded = !children.hidden;
        this.setFolded(folded, toggle, children);
        const paths = readFoldedPaths();
        if (folded) {
          paths.add(nodePath);
        } else {
          paths.delete(nodePath);
        }
        writeFoldedPaths(paths);
      });
    }

    setFolded(folded, toggle, children) {
      children.hidden = folded;
      toggle.setAttribute("aria-expanded", String(!folded));
    }
  }

  function syncCurrentFile() {
    const currentLocation = new URL(window.location.href);
    const currentFile = currentLocation.searchParams.get("file");
    for (const link of document.querySelectorAll("a.tree-file-link")) {
      const linkLocation = new URL(link.href, currentLocation.href);
      const linkedFile = linkLocation.searchParams.get("file");
      const selected = linkedFile === null
        ? linkLocation.pathname === currentLocation.pathname
        : linkedFile === currentFile;
      if (selected) {
        link.setAttribute("aria-current", "page");
      } else {
        link.removeAttribute("aria-current");
      }
    }

    const currentFolder = currentLocation.searchParams.get("folder") || "";
    for (const link of document.querySelectorAll(".file-nav > a")) {
      const linkLocation = new URL(link.href, currentLocation.href);
      const linkedFolder = linkLocation.searchParams.get("folder");
      const currentPath = currentFile || currentFolder;
      const selected = linkedFolder
        ? currentPath === linkedFolder || currentPath.startsWith(`${linkedFolder}/`)
        : linkLocation.pathname === currentLocation.pathname && !currentFile && !currentFolder;
      if (selected) {
        link.setAttribute("aria-current", "page");
      } else {
        link.removeAttribute("aria-current");
      }
    }
  }

  if (!customElements.get("file-tree-node")) {
    customElements.define("file-tree-node", FileTreeNode);
  }

  document.addEventListener("htmx:afterSettle", syncCurrentFile);
  document.addEventListener("htmx:pushedIntoHistory", syncCurrentFile);
  window.addEventListener("popstate", syncCurrentFile);
  syncCurrentFile();
})();
