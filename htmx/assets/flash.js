(() => {
  document.addEventListener("cuebook:flash", (event) => {
    const flash = document.getElementById("flash-message");
    if (!flash) {
      return;
    }
    flash.querySelector("[data-flash-text]").textContent = event.detail.message;
    flash.hidden = false;
  });

  document.querySelector("[data-flash-dismiss]")?.addEventListener("click", () => {
    document.getElementById("flash-message").hidden = true;
  });
})();
