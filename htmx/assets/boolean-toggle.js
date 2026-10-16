class BooleanToggle extends HTMLElement {
  connectedCallback() {
    this.disconnectedCallback();
    const input = this.querySelector('input[type="hidden"]');
    if (input !== this.input) {
      this.defaultValue = input?.defaultValue;
    }
    this.input = input;
    this.button = this.querySelector("button.boolean-toggle");
    this.form = this.input?.form;
    if (!this.input || !this.button) {
      return;
    }

    this.handleClick = () => {
      const state = this.input.value;
      const allowUnset = this.dataset.optional === "true" || this.dataset.allowUnset === "true";
      this.input.value = state === "true"
        ? "false"
        : state === "false" && allowUnset ? "not_set" : "true";
      this.updateButton();
      this.input.dispatchEvent(new Event("input", { bubbles: true }));
      this.input.dispatchEvent(new Event("change", { bubbles: true }));
    };
    this.handleReset = (event) => {
      queueMicrotask(() => {
        if (!this.isConnected || this.input !== input || event.defaultPrevented) {
          return;
        }
        // Hidden inputs couple value and defaultValue, so retain the original reset value.
        this.input.value = this.defaultValue;
        this.updateButton();
      });
    };
    this.button.addEventListener("click", this.handleClick);
    this.form?.addEventListener("reset", this.handleReset);
    this.updateButton();
  }

  disconnectedCallback() {
    this.button?.removeEventListener("click", this.handleClick);
    this.form?.removeEventListener("reset", this.handleReset);
  }

  updateButton() {
    const state = this.input.value === "true" || this.input.value === "false" ? this.input.value : "not_set";
    this.input.value = state;
    this.button.textContent = state === "true" ? "✔" : state === "false" ? "✖" : "☐";
    this.button.classList.toggle("is-link", state === "true");
    this.button.setAttribute("aria-pressed", state === "not_set" ? "mixed" : state);
    this.button.title = `${this.button.getAttribute("aria-label")}: ${state}`;
  }
}

customElements.define("boolean-toggle", BooleanToggle);
