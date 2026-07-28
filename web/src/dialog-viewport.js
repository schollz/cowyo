export const DIALOG_VIEWPORT_PADDING = 16;

const DIALOG_CENTER_PROPERTY = "--crypto-dialog-viewport-center";
const DIALOG_MAX_HEIGHT_PROPERTY = "--crypto-dialog-viewport-max-height";

export function dialogVisualViewportLayout(
  viewport,
  padding = DIALOG_VIEWPORT_PADDING,
) {
  if (
    !viewport ||
    !Number.isFinite(viewport.height) ||
    viewport.height <= 0
  ) {
    return undefined;
  }

  const offsetTop = Number.isFinite(viewport.offsetTop)
    ? viewport.offsetTop
    : 0;
  const safePadding = Number.isFinite(padding) ? Math.max(0, padding) : 0;

  return {
    center: offsetTop + viewport.height / 2,
    maxHeight: Math.max(0, viewport.height - safePadding * 2),
  };
}

export function positionDialogInVisualViewport(dialog, viewport) {
  if (!dialog.open) {
    return false;
  }

  const layout = dialogVisualViewportLayout(viewport);
  if (!layout) {
    return false;
  }

  dialog.style.setProperty(DIALOG_CENTER_PROPERTY, `${layout.center}px`);
  dialog.style.setProperty(
    DIALOG_MAX_HEIGHT_PROPERTY,
    `${layout.maxHeight}px`,
  );
  return true;
}

export function resetDialogVisualViewportPosition(dialog) {
  dialog.style.removeProperty(DIALOG_CENTER_PROPERTY);
  dialog.style.removeProperty(DIALOG_MAX_HEIGHT_PROPERTY);
}
