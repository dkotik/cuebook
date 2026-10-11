(() => {
  const optionSelector = '.file-picker-option input[type="radio"]';

  function getSelectedFile(dialog) {
    return dialog.querySelector(`${optionSelector}:checked:not(:disabled)`);
  }

  function updateConfirm(dialog) {
    const confirm = dialog.querySelector("[data-file-picker-confirm]");
    if (confirm) {
      confirm.disabled = !getSelectedFile(dialog);
    }
  }

  function restorePicker(dialog) {
    const input = dialog.dataset.valueInput
      ? document.getElementById(dialog.dataset.valueInput)
      : null;
    let matched = false;
    for (const option of dialog.querySelectorAll(optionSelector)) {
      option.checked = Boolean(input && !matched && !option.disabled && option.value === input.value);
      matched ||= option.checked;
    }
    updateConfirm(dialog);

    // Reveal the committed file without changing the user's saved folder state.
    let children = getSelectedFile(dialog)?.closest(".tree-children");
    while (children) {
      children.hidden = false;
      children.parentElement.querySelector(":scope > .tree-node-toggle")?.setAttribute("aria-expanded", "true");
      children = children.parentElement.closest(".tree-children");
    }
  }

  document.addEventListener("click", (event) => {
    const target = event.target;
    if (!(target instanceof Element)) {
      return;
    }

    const opener = target.closest("[data-file-picker-open]");
    if (opener) {
      const dialog = document.getElementById(opener.dataset.filePickerOpen);
      if (!(dialog instanceof HTMLDialogElement) || !dialog.matches("dialog[data-file-picker-dialog]") || dialog.open) {
        return;
      }
      restorePicker(dialog);
      dialog.showModal();
      return;
    }

    const dismiss = target.closest("[data-file-picker-dismiss]");
    if (dismiss) {
      dismiss.closest("dialog[data-file-picker-dialog]")?.close();
      return;
    }

    const confirm = target.closest("[data-file-picker-confirm]");
    if (!confirm || confirm.disabled) {
      return;
    }
    const dialog = confirm.closest("dialog[data-file-picker-dialog]");
    const selected = dialog && getSelectedFile(dialog);
    if (!dialog || !selected || !dialog.dataset.valueInput) {
      return;
    }
    const input = document.getElementById(dialog.dataset.valueInput);
    const label = document.getElementById(dialog.dataset.valueLabel);
    if (!input || !label) {
      return;
    }
    input.value = selected.value;
    label.textContent = selected.value;
    dialog.close();
  });

  document.addEventListener("change", (event) => {
    const target = event.target;
    if (!(target instanceof Element) || !target.matches(optionSelector)) {
      return;
    }
    const dialog = target.closest("dialog[data-file-picker-dialog]");
    if (dialog) {
      updateConfirm(dialog);
    }
  });

  document.addEventListener("submit", (event) => {
    const form = event.target;
    if (form instanceof HTMLFormElement) {
      form.closest("dialog[data-file-picker-dialog]")?.close();
    }
  }, true);
})();
