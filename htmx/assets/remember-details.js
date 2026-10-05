(() => {
  const storagePrefix = "cuebook-remember-details:";

  class RememberDetails extends HTMLElement {
    static get observedAttributes() {
      return ["open"];
    }

    connectedCallback() {
      if (this.initialized) {
        return;
      }

      this.summary = this.querySelector(":scope > summary");
      this.content = this.querySelector(":scope > [data-details-content]");
      const stateKey = this.dataset.storageKey;
      if (!this.summary || !this.content || !stateKey) {
        return;
      }

      this.initialized = true;
      this.setAttribute("role", "group");
      this.summary.setAttribute("role", "button");
      this.summary.tabIndex = 0;
      if (this.content.id) {
        this.summary.setAttribute("aria-controls", this.content.id);
      }
      this.storageKey = `${storagePrefix}${stateKey}`;

      this.summary.addEventListener("click", () => this.toggleOpen());
      this.summary.addEventListener("keydown", (event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          this.toggleOpen();
        }
      });

      this.restoring = true;
      try {
        const savedState = window.localStorage.getItem(this.storageKey);
        if (savedState === "open" || savedState === "closed") {
          this.open = savedState === "open";
        }
      } catch (_) {
        // The disclosure remains usable when browser storage is unavailable.
      }
      this.restoring = false;
      this.syncState();
    }

    attributeChangedCallback(name, previousValue, nextValue) {
      if (name !== "open" || previousValue === nextValue) {
        return;
      }
      this.syncState();
      if (this.initialized && !this.restoring) {
        this.persistState();
        this.dispatchEvent(new Event("toggle"));
      }
    }

    get open() {
      return this.hasAttribute("open");
    }

    set open(value) {
      this.toggleAttribute("open", Boolean(value));
    }

    toggleOpen() {
      this.open = !this.open;
    }

    syncState() {
      if (!this.summary || !this.content) {
        return;
      }
      this.summary.setAttribute("aria-expanded", String(this.open));
      this.content.hidden = !this.open;
    }

    persistState() {
      try {
        window.localStorage.setItem(this.storageKey, this.open ? "open" : "closed");
      } catch (_) {
        // The disclosure remains interactive when browser storage is unavailable.
      }
    }
  }

  if (!customElements.get("remember-details")) {
    customElements.define("remember-details", RememberDetails);
  }
})();
