const MIRRORED_TEXT_STYLES = [
  "font",
  "letterSpacing",
  "lineHeight",
  "overflowWrap",
  "padding",
  "tabSize",
  "textAlign",
  "textIndent",
  "textTransform",
  "whiteSpace",
  "wordBreak",
  "wordSpacing",
];

function clamp(value, minimum, maximum) {
  return Math.min(Math.max(value, minimum), maximum);
}

export function calculateAnchoredStatusPosition({
  anchorLeft,
  anchorTop,
  anchorBottom,
  statusWidth,
  statusHeight,
  viewportWidth,
  viewportHeight,
  gap = 10,
  margin = 16,
}) {
  const maximumLeft = Math.max(margin, viewportWidth - statusWidth - margin);
  const left = clamp(anchorLeft - 24, margin, maximumLeft);
  const pointerLeft = clamp(
    anchorLeft - left - 7,
    12,
    Math.max(12, statusWidth - 26),
  );
  const belowTop = anchorBottom + gap;
  const aboveTop = anchorTop - statusHeight - gap;
  const canFitBelow = belowTop + statusHeight <= viewportHeight - margin;
  const canFitAbove = aboveTop >= margin;
  const placement = canFitBelow || !canFitAbove ? "below" : "above";
  const requestedTop = placement === "below" ? belowTop : aboveTop;
  const maximumTop = Math.max(margin, viewportHeight - statusHeight - margin);

  return {
    left,
    placement,
    pointerLeft,
    top: clamp(requestedTop, margin, maximumTop),
  };
}

export function measureTextareaEnd(textarea) {
  const documentObject = textarea.ownerDocument;
  const view = documentObject.defaultView;
  const computedStyle = view.getComputedStyle(textarea);
  const mirror = documentObject.createElement("div");
  const marker = documentObject.createElement("span");

  mirror.style.position = "fixed";
  mirror.style.top = "0";
  mirror.style.left = "-100000px";
  mirror.style.width = `${textarea.clientWidth}px`;
  mirror.style.height = "auto";
  mirror.style.margin = "0";
  mirror.style.border = "0";
  mirror.style.boxSizing = "border-box";
  mirror.style.overflow = "visible";
  mirror.style.visibility = "hidden";
  mirror.style.pointerEvents = "none";
  for (const property of MIRRORED_TEXT_STYLES) {
    mirror.style[property] = computedStyle[property];
  }

  mirror.append(documentObject.createTextNode(textarea.value));
  marker.textContent = "\u200b";
  mirror.append(marker);
  documentObject.body.append(mirror);

  const textareaRect = textarea.getBoundingClientRect();
  const mirrorRect = mirror.getBoundingClientRect();
  const markerRect = marker.getBoundingClientRect();
  const lineHeight =
    markerRect.height ||
    Number.parseFloat(computedStyle.lineHeight) ||
    Number.parseFloat(computedStyle.fontSize) * 1.2;
  const left =
    textareaRect.left + markerRect.left - mirrorRect.left - textarea.scrollLeft;
  const top =
    textareaRect.top + markerRect.top - mirrorRect.top - textarea.scrollTop;

  mirror.remove();
  return { bottom: top + lineHeight, left, top };
}
