export const PRIVATE_ACTIVE_STATUS =
  "Private page active. Copy the complete #key URL; anyone with it can read and edit, and lost keys cannot be recovered.";
export const PRIVATE_KEY_ERROR_STATUS =
  "This private link has a missing or malformed key.";

export function createPrivateStatusController(container, messageElement) {
  let activeStatusDismissed = false;

  function dismiss() {
    if (messageElement.textContent === PRIVATE_ACTIVE_STATUS) {
      activeStatusDismissed = true;
    }
    container.hidden = true;
  }

  function show(message, error = false) {
    const active = message === PRIVATE_ACTIVE_STATUS;
    const keyError = message === PRIVATE_KEY_ERROR_STATUS;
    container.dataset.active = String(active);
    container.dataset.anchored = String(active || keyError);
    container.dataset.keyError = String(keyError);
    if (active && activeStatusDismissed) {
      container.hidden = true;
      return;
    }

    messageElement.textContent = message;
    container.dataset.error = String(error);
    container.hidden = !message;
  }

  return {
    show,
    showActiveWhenEmpty(content) {
      if (content.length !== 0) {
        container.hidden = true;
        return false;
      }
      show(PRIVATE_ACTIVE_STATUS);
      return !container.hidden;
    },
    hideActiveWhenContent(content) {
      if (
        content.length === 0 ||
        messageElement.textContent !== PRIVATE_ACTIVE_STATUS
      ) {
        return false;
      }
      dismiss();
      return true;
    },
    dismiss,
    dismissWhenOutside(target) {
      if (container.hidden || container.contains(target)) {
        return false;
      }
      dismiss();
      return true;
    },
  };
}
