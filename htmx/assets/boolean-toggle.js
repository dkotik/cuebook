class BooleanToggle extends HTMLElement {
  static get observedAttributes() {
    return ["expanded"];
  }

  connectedCallback() {
    this.disconnectedCallback();
    const input = this.querySelector('input[type="hidden"]');
    if (input !== this.input) {
      this.defaultValue = input?.defaultValue;
    }
    this.input = input;
    const button = this.querySelector("button.boolean-toggle");
    if (button !== this.button) {
      this.buttonLabel = button?.getAttribute("aria-label") || "";
    }
    this.button = button;
    this.form = this.input?.form;
    if (!this.input || !this.button) {
      this.buttons = [];
      return;
    }

    this.handleClick = (event) => {
      const clickedButton = event.currentTarget || event.target;
      if (this.hasAttribute("expanded")) {
        const selectedState = clickedButton.dataset.booleanState;
        if (!["true", "false"].includes(selectedState)) {
          return;
        }
        this.input.value = this.input.value === selectedState ? "not_set" : selectedState;
      } else {
        const state = this.input.value;
        const allowUnset = this.dataset.optional === "true" || this.dataset.allowUnset === "true";
        this.input.value = state === "true"
          ? "false"
          : state === "false" && allowUnset ? "not_set" : "true";
      }
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
    this.configureButtons();
    this.buttons.forEach((button) => button.addEventListener("click", this.handleClick));
    this.form?.addEventListener("reset", this.handleReset);
    this.updateButton();
  }

  attributeChangedCallback(name) {
    if (name !== "expanded" || !this.input || !this.button || !this.handleClick) {
      return;
    }
    this.buttons.forEach((button) => button.removeEventListener("click", this.handleClick));
    this.configureButtons();
    this.buttons.forEach((button) => button.addEventListener("click", this.handleClick));
    this.updateButton();
  }

  disconnectedCallback() {
    this.buttons?.forEach((button) => button.removeEventListener("click", this.handleClick));
    this.form?.removeEventListener("reset", this.handleReset);
  }

  configureButtons() {
    if (this.hasAttribute("expanded")) {
      this.setAttribute("role", "group");
      this.setAttribute("aria-label", this.buttonLabel);
      this.button.dataset.booleanState = "true";
      const existing = new Map((this.expandedButtons || []).map((button) => [button.dataset.booleanState, button]));
      this.expandedButtons = ["false"].map((state) => {
        const button = existing.get(state) || this.button.cloneNode(false);
        button.dataset.booleanState = state;
        if (button.parentElement !== this) {
          this.appendChild(button);
        }
        return button;
      });
      this.buttons = [this.button, ...this.expandedButtons];
      return;
    }

    this.expandedButtons?.forEach((button) => {
      if (button.parentElement === this) {
        this.removeChild(button);
      }
    });
    this.expandedButtons = [];
    this.button.removeAttribute("data-boolean-state");
    this.removeAttribute("role");
    this.removeAttribute("aria-label");
    this.buttons = [this.button];
  }

  updateButton() {
    const state = this.input.value === "true" || this.input.value === "false" ? this.input.value : "not_set";
    this.input.value = state;
    if (this.hasAttribute("expanded")) {
      const symbols = { true: "✔", false: "✖" };
      for (const button of this.buttons) {
        const buttonState = button.dataset.booleanState;
        const selected = buttonState === state;
        button.classList.toggle("is-link", selected && buttonState === "true");
        button.classList.toggle("has-text-info", selected && buttonState === "true");
        button.classList.toggle("has-text-danger", selected && buttonState === "false");
        button.textContent = symbols[buttonState];
        button.setAttribute("aria-pressed", String(selected));
        const label = `${this.buttonLabel}: ${buttonState}`;
        button.setAttribute("aria-label", label);
        button.title = label;
      }
      return;
    }

    this.button.classList.toggle("is-link", state === "true");
    this.button.classList.toggle("has-text-info", state === "true");
    this.button.classList.toggle("has-text-danger", state === "false");
    switch (state) {
      case "true":
        this.button.textContent = "✔";
        break;
      case "false":
        this.button.textContent = "✖";
        break;
      default:
        this.button.textContent = "☐";
    }
    this.button.setAttribute("aria-label", this.buttonLabel);
    this.button.setAttribute("aria-pressed", state === "not_set" ? "mixed" : state);
    this.button.title = `${this.buttonLabel}: ${state}`;
  }
}

customElements.define("boolean-toggle", BooleanToggle);
