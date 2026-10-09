(() => {
  document.addEventListener("paste", (event) => {
    const target = event.target;
    if (target instanceof Element &&
        (target.closest("input, textarea, select") || target.isContentEditable)) {
      return;
    }
    const destination = document.querySelector("#workspace [data-paste-file]");
    if (!destination || !event.clipboardData) {
      return;
    }

    event.preventDefault();
    const workspace = document.getElementById("workspace");
    window.htmx.ajax("POST", destination.dataset.pasteUrl, {
      source: workspace,
      target: workspace,
      swap: "outerHTML",
      values: {
        file: destination.dataset.pasteFile,
        source: event.clipboardData.getData("text/plain"),
      },
    }).catch(() => {
      window.htmx.trigger(document.body, "cuebook:flash", {
        message: "The pasted entry could not be sent. Please try again.",
      });
    });
  });
})();
