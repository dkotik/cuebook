class CuebookSearch extends HTMLElement {
  connectedCallback() {
    this.input = this.querySelector('input[type="search"][name="q"]');
    this.clearButton = this.querySelector("[data-clear-search]");
    if (!this.input || !this.clearButton) {
      return;
    }

    this.handleInput = () => this.updateClearButton();
    this.handleClear = () => {
      this.input.value = "";
      this.updateClearButton();
      this.input.focus();
    };
    this.input.addEventListener("input", this.handleInput);
    this.clearButton.addEventListener("click", this.handleClear);
    this.updateClearButton();
  }

  disconnectedCallback() {
    this.input?.removeEventListener("input", this.handleInput);
    this.clearButton?.removeEventListener("click", this.handleClear);
  }

  updateClearButton() {
    if (this.input && this.clearButton) {
      this.clearButton.hidden = this.input.value.trim() === "";
    }
  }
}

customElements.define("cuebook-search", CuebookSearch);
