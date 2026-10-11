class CuebookSearch extends HTMLElement {
  connectedCallback() {
    this.form = this.querySelector("form.topbar-search");
    this.input = this.querySelector('input[type="search"][name="q"]');
    this.inputControl = this.input?.closest(".control");
    this.clearControl ||= this.querySelector("template[data-clear-search-template]")?.content.firstElementChild;
    this.clearButton ||= this.clearControl?.querySelector("[data-clear-search]");
    if (!this.form || !this.input || !this.inputControl || !this.clearControl || !this.clearButton) {
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
    if (!this.input || !this.inputControl || !this.form || !this.clearControl) {
      return;
    }

    const hasQuery = this.input.value.trim() !== "";
    if (hasQuery) {
      if (this.clearControl.parentElement !== this.form || this.inputControl.previousElementSibling !== this.clearControl) {
        this.form.insertBefore(this.clearControl, this.inputControl);
      }
    } else {
      this.clearControl.remove();
    }
  }
}

customElements.define("cuebook-search", CuebookSearch);
