export const SAVE_ACTIVITY_IDLE_DELAY = 1000;

export function createSaveActivityIndicator(
  element,
  {
    setTimeout = globalThis.setTimeout,
    clearTimeout = globalThis.clearTimeout,
  } = {},
) {
  let idleTimer;

  return function markSaveActivity() {
    if (idleTimer !== undefined) {
      clearTimeout(idleTimer);
    }
    element.classList.add("is-save-active");
    idleTimer = setTimeout(() => {
      element.classList.remove("is-save-active");
      idleTimer = undefined;
    }, SAVE_ACTIVITY_IDLE_DELAY);
  };
}
