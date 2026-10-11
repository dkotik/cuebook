class CuebookSearch extends HTMLElement {
  connectedCallback() {
    this.input = this.querySelector('input[type="search"][name="q"]');
    this.clearButton = this.querySelector("[data-clear-search]");
    this.submitButton = this.querySelector('button[type="submit"]');
    if (!this.input || !this.clearButton) {
      return;
    }

    this.handleInput = () => this.updateButtons();
    this.handleClear = () => {
      this.input.value = "";
      this.updateButtons();
      this.input.focus();
    };
    this.input.addEventListener("input", this.handleInput);
    this.clearButton.addEventListener("click", this.handleClear);
    this.updateButtons();
  }

  disconnectedCallback() {
    this.input?.removeEventListener("input", this.handleInput);
    this.clearButton?.removeEventListener("click", this.handleClear);
  }

  updateButtons() {
    if (this.input && this.clearButton) {
      const hasQuery = this.input.value.trim() !== "";
      this.clearButton.hidden = !hasQuery;
    }
  }
}

customElements.define("cuebook-search", CuebookSearch);
