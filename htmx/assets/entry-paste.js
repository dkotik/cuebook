(() => {
  const cutKey = "cuebook-cut:" + (document.body?.dataset.routePrefix || "/");
  let pending = false;

  document.addEventListener("paste", async (event) => {
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
    if (pending) {
      return;
    }
    const workspace = document.getElementById("workspace");
    const values = {
      file: destination.dataset.pasteFile,
      source: event.clipboardData.getData("text/plain"),
    };
    let storedCut;
    try {
      storedCut = localStorage.getItem(cutKey);
      if (storedCut) {
        const cut = JSON.parse(storedCut);
        if (!cut || typeof cut.file !== "string" || typeof cut.entry !== "string" || typeof cut.fingerprint !== "string") {
          throw new Error("Invalid cut record");
        }
        values.cut_file = cut.file;
        values.cut_entry = cut.entry;
        values.cut_fingerprint = cut.fingerprint;
      }
    } catch (_) {
      window.htmx.trigger(document.body, "cuebook:flash", {
        message: "Unable to read the cut status. Copy or cut the entry again before pasting.",
      });
      return;
    }

    let succeeded = false;
    const afterRequest = (requestEvent) => {
      const detail = requestEvent.detail;
      const parameters = detail?.requestConfig?.parameters;
      if (detail?.requestConfig?.path === destination.dataset.pasteUrl &&
          parameters?.cut_file === values.cut_file &&
          parameters?.cut_entry === values.cut_entry &&
          parameters?.cut_fingerprint === values.cut_fingerprint) {
        succeeded = detail.xhr.status === 200;
      }
    };
    if (storedCut) {
      document.addEventListener("htmx:afterRequest", afterRequest);
    }
    pending = true;
    try {
      await window.htmx.ajax("POST", destination.dataset.pasteUrl, {
        source: workspace,
        target: workspace,
        swap: "outerHTML",
        values,
      });
      if (storedCut && succeeded) {
        try {
          if (localStorage.getItem(cutKey) === storedCut) {
            localStorage.removeItem(cutKey);
          }
        } catch (_) {
          window.htmx.trigger(document.body, "cuebook:flash", {
            message: "The entry was pasted, but its cut status could not be cleared. Copy before pasting again.",
          });
        }
      }
    } catch (_) {
      window.htmx.trigger(document.body, "cuebook:flash", {
        message: "The pasted entry could not be sent. Please try again.",
      });
    } finally {
      pending = false;
      document.removeEventListener("htmx:afterRequest", afterRequest);
    }
  });
})();
