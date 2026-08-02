export const PRIVATE_ACTIVE_STATUS =
  "Private page active. Copy the complete #key URL; anyone with it can read and edit, and lost keys cannot be recovered.";

export function createPrivateStatusController(container, messageElement) {
  let activeStatusDismissed = false;

  return {
    show(message, error = false) {
      if (message === PRIVATE_ACTIVE_STATUS && activeStatusDismissed) {
        container.hidden = true;
        return;
      }

      messageElement.textContent = message;
      container.dataset.error = String(error);
      container.hidden = !message;
    },
    dismiss() {
      if (messageElement.textContent === PRIVATE_ACTIVE_STATUS) {
        activeStatusDismissed = true;
      }
      container.hidden = true;
    },
  };
}
