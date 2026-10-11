(() => {
  function getSelectedFile(dialog) {
    return dialog.querySelector('input[type="radio"]:checked:not(:disabled)');
  }

  function syncPicker(dialog) {
    const input = document.getElementById(dialog.dataset.valueInput);
    if (input) {
      let matched = false;
      for (const option of dialog.querySelectorAll('input[type="radio"]')) {
        const selected = !matched && !option.disabled && option.value === input.value;
        option.checked = selected;
        matched ||= selected;
      }
    }
    const confirm = dialog.querySelector("[data-file-picker-confirm]");
    if (confirm) {
      confirm.disabled = !getSelectedFile(dialog);
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
      if (!(dialog instanceof HTMLDialogElement) || dialog.open) {
        return;
      }
      syncPicker(dialog);
      dialog.showModal();
      return;
    }

    const dismiss = target.closest("[data-file-picker-dismiss]");
    if (dismiss) {
      dismiss.closest("dialog")?.close();
      return;
    }

    const confirm = target.closest("[data-file-picker-confirm]");
    if (!confirm) {
      return;
    }
    const dialog = confirm.closest("dialog[data-file-picker-dialog]");
    const selected = dialog && getSelectedFile(dialog);
    if (!dialog || !selected) {
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
    const dialog = target instanceof Element ? target.closest("dialog[data-file-picker-dialog]") : null;
    if (dialog && target.matches('input[type="radio"]')) {
      syncPicker(dialog);
    }
  });
})();
