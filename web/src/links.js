export function clickableURL(line) {
  const value = line.trim();
  if (value === "" || /\s/.test(value)) {
    return null;
  }

  try {
    const url = new URL(value);
    if (url.protocol !== "http:" && url.protocol !== "https:") {
      return null;
    }
    return url.href;
  } catch {
    return null;
  }
}

function trimTrailingPunctuation(value) {
  while (/[.,!?;:]$/.test(value)) {
    value = value.slice(0, -1);
  }

  [
    ["(", ")"],
    ["[", "]"],
    ["{", "}"],
  ].forEach(([opening, closing]) => {
    const openingCount = value.split(opening).length - 1;
    let closingCount = value.split(closing).length - 1;

    while (value.endsWith(closing) && closingCount > openingCount) {
      value = value.slice(0, -1);
      closingCount--;
    }
  });

  return value;
}

export function findURLs(text) {
  const matches = [];
  const pattern = /https?:\/\/[^\s<>"'`]+/gi;
  let match;

  while ((match = pattern.exec(text)) !== null) {
    const value = trimTrailingPunctuation(match[0]);
    const href = clickableURL(value);

    if (href !== null) {
      matches.push({
        start: match.index,
        end: match.index + value.length,
        text: value,
        href,
      });
    }
  }

  return matches;
}

export function renderLinks(textarea, overlay) {
  overlay.replaceChildren();

  const text = textarea.value;
  let position = 0;

  findURLs(text).forEach((match) => {
    overlay.append(document.createTextNode(text.slice(position, match.start)));

    const anchor = document.createElement("a");
    anchor.href = match.href;
    anchor.target = "_blank";
    anchor.rel = "noopener noreferrer";
    anchor.textContent = match.text;
    overlay.append(anchor);

    position = match.end;
  });

  overlay.append(document.createTextNode(text.slice(position)));
  overlay.scrollTop = textarea.scrollTop;
  overlay.scrollLeft = textarea.scrollLeft;
}
