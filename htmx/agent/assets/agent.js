(() => {
  const storageKey = "cuebook-agent-window";

  class CuebookAgent extends HTMLElement {
    connectedCallback() {
      if (this.dataset.agentReady === "true") return;
      this.dataset.agentReady = "true";
      this.window = this.querySelector(".agent-window");
      this.launcher = this.querySelector(".agent-launcher");
      this.minimizeButton = this.querySelector(".agent-minimize");
      this.titlebar = this.querySelector("[data-agent-drag-handle]");
      this.transcript = this.querySelector(".agent-transcript");
      this.restore();
      if (this.window && typeof ResizeObserver !== "undefined") {
        this.resizeObserver = new ResizeObserver(() => {
          if (!this.window.hidden) this.persist();
        });
        this.resizeObserver.observe(this.window);
      }

      this.launcher?.addEventListener("click", () => this.setMinimized(false));
      this.minimizeButton?.addEventListener("click", () => this.setMinimized(true));
      this.querySelectorAll("[data-agent-dock]").forEach((button) => {
        button.addEventListener("click", () => this.dock(button.dataset.agentDock));
      });
      this.titlebar?.addEventListener("pointerdown", (event) => this.startDrag(event));
      this.addEventListener("htmx:beforeRequest", () => this.window?.setAttribute("aria-busy", "true"));
      this.addEventListener("htmx:afterRequest", (event) => {
        this.window?.removeAttribute("aria-busy");
        if (!event.detail?.successful) return;
        this.transcript?.scrollTo({ top: this.transcript.scrollHeight, behavior: "smooth" });
        const input = this.querySelector(".agent-message-input");
        if (input) input.value = "";
      });
      this.addEventListener("htmx:responseError", () => this.window?.removeAttribute("aria-busy"));
    }

    setMinimized(minimized) {
      this.dataset.minimized = String(minimized);
      if (this.window) this.window.hidden = minimized;
      this.launcher?.setAttribute("aria-expanded", String(!minimized));
      this.minimizeButton?.setAttribute("aria-expanded", String(!minimized));
      if (!minimized) this.querySelector(".agent-message-input")?.focus();
      this.persist();
    }

    dock(side) {
      this.dataset.dock = side === "left" ? "left" : "right";
      this.style.left = "";
      this.style.right = "";
      this.style.top = "";
      this.style.bottom = "1rem";
      this.persist();
    }

    startDrag(event) {
      if (event.button !== 0 || event.target.closest("button")) return;
      event.preventDefault();
      const bounds = this.getBoundingClientRect();
      const offsetX = event.clientX - bounds.left;
      const offsetY = event.clientY - bounds.top;
      this.setPointerCapture(event.pointerId);

      const move = (next) => {
        const maxLeft = Math.max(0, window.innerWidth - this.offsetWidth);
        const maxTop = Math.max(0, window.innerHeight - this.offsetHeight);
        const left = Math.min(maxLeft, Math.max(0, next.clientX - offsetX));
        const top = Math.min(maxTop, Math.max(0, next.clientY - offsetY));
        this.dataset.dock = left + this.offsetWidth / 2 < window.innerWidth / 2 ? "left" : "right";
        this.style.left = `${left}px`;
        this.style.right = "auto";
        this.style.top = `${top}px`;
        this.style.bottom = "auto";
      };
      const finish = () => {
        this.removeEventListener("pointermove", move);
        this.removeEventListener("pointerup", finish);
        this.removeEventListener("pointercancel", finish);
        this.persist();
      };
      this.addEventListener("pointermove", move);
      this.addEventListener("pointerup", finish, { once: true });
      this.addEventListener("pointercancel", finish, { once: true });
    }

    persist() {
      try {
        localStorage.setItem(storageKey, JSON.stringify({
          minimized: this.dataset.minimized === "true",
          dock: this.dataset.dock || "right",
          left: this.style.left,
          top: this.style.top,
          width: this.window ? `${Math.round(this.window.getBoundingClientRect().width)}px` : "",
          height: this.window ? `${Math.round(this.window.getBoundingClientRect().height)}px` : "",
        }));
      } catch (_) {
        // Storage can be disabled by browser privacy settings.
      }
    }

    restore() {
      try {
        const saved = JSON.parse(localStorage.getItem(storageKey) || "null");
        if (!saved) {
          this.setMinimized(true);
          return;
        }
        this.dataset.dock = saved.dock === "left" ? "left" : "right";
        this.dataset.minimized = String(Boolean(saved.minimized));
        if (this.window) {
          this.window.hidden = Boolean(saved.minimized);
          this.window.style.width = saved.width || "";
          this.window.style.height = saved.height || "";
        }
        if (saved.left && saved.top && window.matchMedia("(min-width: 40.01rem)").matches) {
          this.style.left = saved.left;
          this.style.right = "auto";
          this.style.top = saved.top;
          this.style.bottom = "auto";
        }
        this.launcher?.setAttribute("aria-expanded", String(!saved.minimized));
        this.minimizeButton?.setAttribute("aria-expanded", String(!saved.minimized));
      } catch (_) {
        this.dataset.minimized = "true";
      }
    }
  }

  if (!customElements.get("cuebook-agent")) {
    customElements.define("cuebook-agent", CuebookAgent);
  }
})();
