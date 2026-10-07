class LiveReload extends HTMLElement {
  connectedCallback() {
    if (this.eventSource || this.reloadPending) return;

    const eventsURL = this.dataset.eventsUrl || "/events";
    const eventSource = new EventSource(eventsURL);
    this.eventSource = eventSource;
    eventSource.addEventListener("error", () => {
      eventSource.close();
      this.eventSource = null;
      if (this.reloadPending) return;

      this.reloadPending = true;
      void this.waitForServer();
    }, { once: true });
  }

  waitForServer() {
    window.clearTimeout(this.reloadTimeout);
    this.reloadTimeout = window.setTimeout(async () => {
      this.reloadTimeout = null;
      if (!this.isConnected) return;

      try {
        await fetch("/", { cache: "no-store" });
        if (this.isConnected) window.location.reload();
      } catch {
        if (this.isConnected) this.waitForServer();
      }
    }, 2000);
  }

  disconnectedCallback() {
    this.eventSource?.close();
    this.eventSource = null;
    window.clearTimeout(this.reloadTimeout);
    this.reloadTimeout = null;
  }
}

customElements.define("live-reload", LiveReload);
