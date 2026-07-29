export function clampRemoteCursorPosition(position, textLength) {
  const numericPosition = Number(position);
  if (!Number.isFinite(numericPosition)) {
    return undefined;
  }
  return Math.min(
    textLength,
    Math.max(0, Math.trunc(numericPosition)),
  );
}

export function createCursorBroadcastGuard(timers = globalThis) {
  let resumeTimer;
  let paused = false;

  return {
    canBroadcast() {
      return !paused;
    },
    pause() {
      timers.clearTimeout(resumeTimer);
      resumeTimer = undefined;
      paused = true;
    },
    resumeAfterCurrentTask() {
      timers.clearTimeout(resumeTimer);
      resumeTimer = timers.setTimeout(() => {
        paused = false;
        resumeTimer = undefined;
      }, 0);
    },
  };
}

export function cursorPositionChanged(previous, current) {
  return (
    previous === undefined ||
    previous.start !== current.start ||
    previous.end !== current.end
  );
}

export function createRemoteCursorOverlay(textarea, overlay) {
  const snapshots = new Map();
  let textChangeActive = false;
  let ignoredTextChangeScroll;

  function currentScroll() {
    return {
      top: textarea.scrollTop,
      left: textarea.scrollLeft,
    };
  }

  function setSnapshotScroll(snapshot) {
    snapshot.scrollTop = textarea.scrollTop;
    snapshot.scrollLeft = textarea.scrollLeft;
  }

  return {
    beginTextChange() {
      textChangeActive = true;
    },
    finishTextChange() {
      textChangeActive = false;
      ignoredTextChangeScroll = currentScroll();
    },
    clear() {
      snapshots.clear();
      overlay.replaceChildren();
    },
    remove(clientId) {
      const snapshot = snapshots.get(clientId);
      if (!snapshot) {
        return;
      }
      snapshot.remove();
      snapshots.delete(clientId);
    },
    syncScroll() {
      const scroll = currentScroll();
      if (textChangeActive) {
        ignoredTextChangeScroll = scroll;
        return false;
      }
      if (
        ignoredTextChangeScroll &&
        ignoredTextChangeScroll.top === scroll.top &&
        ignoredTextChangeScroll.left === scroll.left
      ) {
        ignoredTextChangeScroll = undefined;
        return false;
      }
      ignoredTextChangeScroll = undefined;
      snapshots.forEach(setSnapshotScroll);
      return true;
    },
    update(clientId, position) {
      if (!clientId) {
        return;
      }

      let snapshot = snapshots.get(clientId);
      if (!snapshot) {
        snapshot = overlay.ownerDocument.createElement("div");
        snapshot.className = "remote-cursor-snapshot";
        snapshot.dataset.clientId = clientId;
        snapshot.setAttribute("aria-hidden", "true");
        overlay.append(snapshot);
        snapshots.set(clientId, snapshot);
      }
      renderRemoteCursorSnapshot(
        textarea,
        snapshot,
        clientId,
        position,
      );
    },
  };
}

export function renderRemoteCursorSnapshot(
  textarea,
  snapshot,
  clientId,
  position,
) {
  snapshot.replaceChildren();

  const text = textarea.value;
  const cursorPosition = clampRemoteCursorPosition(
    position,
    text.length,
  );
  if (cursorPosition === undefined) {
    return;
  }

  snapshot.append(
    snapshot.ownerDocument.createTextNode(
      text.slice(0, cursorPosition),
    ),
  );

  const marker = snapshot.ownerDocument.createElement("span");
  marker.className = "remote-cursor";
  marker.dataset.clientId = clientId;
  marker.setAttribute("aria-hidden", "true");
  snapshot.append(marker);

  snapshot.append(
    snapshot.ownerDocument.createTextNode(text.slice(cursorPosition)),
  );
  snapshot.scrollTop = textarea.scrollTop;
  snapshot.scrollLeft = textarea.scrollLeft;
}
