(() => {
  const burger = document.querySelector(".topbar .navbar-burger");
  if (!burger) {
    return;
  }

  const menu = document.getElementById(burger.getAttribute("aria-controls"));
  if (!menu) {
    return;
  }

  function setExpanded(expanded) {
    burger.classList.toggle("is-active", expanded);
    menu.classList.toggle("is-active", expanded);
    burger.setAttribute("aria-expanded", String(expanded));
  }

  burger.addEventListener("click", () => {
    setExpanded(burger.getAttribute("aria-expanded") !== "true");
  });

  document.querySelector(".topbar-search")?.addEventListener("submit", () => setExpanded(false));
  menu.querySelector("#theme-toggle")?.addEventListener("click", () => setExpanded(false));
})();
