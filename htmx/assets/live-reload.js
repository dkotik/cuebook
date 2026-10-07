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
      window.setTimeout(() => window.location.reload(), 2000);
    }, { once: true });
  }

  disconnectedCallback() {
    this.eventSource?.close();
    this.eventSource = null;
  }
}

customElements.define("live-reload", LiveReload);
