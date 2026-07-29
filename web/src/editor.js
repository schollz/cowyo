import {
  Bomb,
  Check,
  Copy,
  createElement,
  createIcons,
  Eye,
  EyeClosed,
  Globe2,
  GlobeLock,
  Info,
  LockKeyhole,
  Moon,
  ShieldCheck,
  ShieldKeyhole,
  Sun,
  UnlockKeyhole,
} from "lucide";

import {
  decryptEncryptedBlocks,
  encryptText,
  hasEncryptedBlocks,
} from "./encryption.js";
import { renderLinks } from "./links.js";
import {
  applySystemTheme,
  applyTheme,
  DARK_THEME_QUERY,
  readStoredTheme,
  storeTheme,
  systemTheme,
} from "./theme.js";
import { createSaveActivityIndicator } from "./save-activity.js";
import {
  passwordVisibilityState,
  validatePasswordFields,
} from "./password-fields.js";
import {
  positionDialogInVisualViewport,
  resetDialogVisualViewportPosition,
} from "./dialog-viewport.js";
import {
  createCursorBroadcastGuard,
  createRemoteCursorOverlay,
  cursorPositionChanged,
} from "./remote-cursors.js";
import {
  cursorMessage,
  editMessage,
  operationMessage,
  websocketMessageType,
} from "./websocket-protocol.js";
const textarea = document.querySelector("textarea");
const editor = document.querySelector(".editor");
const linkOverlay = document.getElementById("linkOverlay");
const cursorOverlay = document.getElementById("cursorOverlay");
const saveActions = document.getElementById("saveActions");
const saveStatus = document.getElementById("saveStatus");
const saveMenu = document.getElementById("saveMenu");
const saveStatusText = document.getElementById("saveStatusText");
const themeAction = document.getElementById("themeAction");
const copyTextAction = document.getElementById("copyTextAction");
const cryptoAction = document.getElementById("cryptoAction");
const publishAction = document.getElementById("publishAction");
const pageLockAction = document.getElementById("pageLockAction");
const selfDestructAction = document.getElementById("selfDestructAction");
const cryptoDialog = document.getElementById("cryptoDialog");
const cryptoDialogDescription = document.getElementById(
  "cryptoDialogDescription",
);
const cryptoForm = document.getElementById("cryptoForm");
const cryptoPasswordField = document.getElementById("cryptoPasswordField");
const cryptoError = document.getElementById("cryptoError");
const cryptoCancel = document.getElementById("cryptoCancel");
const cryptoSubmit = document.getElementById("cryptoSubmit");
const remoteUpdateStatus = document.getElementById("remoteUpdateStatus");

const ICON_ATTRIBUTES = {
  "aria-hidden": "true",
  "stroke-width": 1.8,
};

createIcons({
  icons: {
    Bomb,
    Check,
    Copy,
    Globe2,
    GlobeLock,
    Info,
    LockKeyhole,
    Moon,
    ShieldCheck,
    ShieldKeyhole,
    Sun,
    UnlockKeyhole,
  },
  attrs: ICON_ATTRIBUTES,
});

const cryptoLockIcon = cryptoAction.querySelector("[data-crypto-lock]");
const cryptoUnlockIcon = cryptoAction.querySelector("[data-crypto-unlock]");
const pageLockIcon = pageLockAction.querySelector("[data-page-lock]");
const pageUnlockIcon = pageLockAction.querySelector("[data-page-unlock]");
const publishIcon = publishAction.querySelector("[data-publish]");
const unpublishIcon = publishAction.querySelector("[data-unpublish]");
const copyIcon = copyTextAction.querySelector("[data-copy]");
const copySuccessIcon = copyTextAction.querySelector("[data-copy-success]");
const darkThemeIcon = themeAction.querySelector("[data-theme-dark]");
const lightThemeIcon = themeAction.querySelector("[data-theme-light]");
const themeColorMeta = document.querySelector('meta[name="theme-color"]');
const colorSchemeQuery = window.matchMedia(DARK_THEME_QUERY);
const COPY_FEEDBACK_DELAY = 1500;

const dialogDetails = {
  encrypt: {
    submit: "Encrypt",
    busy: "Encrypting…",
    description: "",
    minLength: 8,
    confirm: true,
  },
  decrypt: {
    submit: "Decrypt",
    busy: "Decrypting…",
    description:
      "Only signed encrypted blocks will be replaced. Other text stays untouched.",
    minLength: 1,
  },
  lock: {
    submit: "Lock",
    busy: "Locking…",
    description:
      "Editing and POSTs will be blocked until this page is unlocked with the same password.",
    minLength: 8,
    confirm: true,
  },
  unlock: {
    submit: "Unlock",
    busy: "Unlocking…",
    description:
      "Enter the page-lock password to allow editing again.",
    minLength: 1,
  },
};

let socket;
let pendingUpdate;
let outstandingSaves = 0;
let reconnectTimer;
let copyFeedbackTimer;
let remoteStatusTimer;
let cryptoPassword;
let cryptoPasswordConfirmation;
let dialogMode = "encrypt";
let dialogBusy = false;
let pendingPageOperation;
let pageLocked = textarea.dataset.pageLocked === "true";
let pagePublished = textarea.dataset.pagePublished === "true";
let pageSelfDestruct = textarea.dataset.pageSelfDestruct === "true";
let selectedTheme = readStoredTheme(window.localStorage);
let lastAnnouncedCursor;
const cursorBroadcastGuard = createCursorBroadcastGuard(window);
const remoteCursorOverlay = createRemoteCursorOverlay(
  textarea,
  cursorOverlay,
);

function updateThemeAction(theme) {
  const isDark = theme === "dark";
  darkThemeIcon.toggleAttribute("hidden", isDark);
  lightThemeIcon.toggleAttribute("hidden", !isDark);
  const label = isDark
    ? "Use light mode on this device"
    : "Use dark mode on this device";
  themeAction.setAttribute("aria-label", label);
  themeAction.dataset.tooltip = label;
}

function syncTheme() {
  let theme;
  if (selectedTheme) {
    applyTheme(document.documentElement, themeColorMeta, selectedTheme);
    theme = selectedTheme;
  } else {
    theme = applySystemTheme(
      document.documentElement,
      themeColorMeta,
      colorSchemeQuery,
    );
  }
  updateThemeAction(theme);
}

function toggleTheme() {
  const activeTheme = selectedTheme || systemTheme(colorSchemeQuery);
  selectedTheme = activeTheme === "dark" ? "light" : "dark";
  storeTheme(window.localStorage, selectedTheme);
  syncTheme();
}

function resetCopyFeedback() {
  window.clearTimeout(copyFeedbackTimer);
  copyFeedbackTimer = undefined;
  copyIcon.hidden = false;
  copySuccessIcon.hidden = true;
  copyTextAction.classList.remove("is-success");
  copyTextAction.setAttribute("aria-label", "Copy paste text");
  copyTextAction.dataset.tooltip = "Copy paste text";
}

function debounce(callback, wait) {
  let timeout;
  const debounced = (...args) => {
    window.clearTimeout(timeout);
    timeout = window.setTimeout(() => callback(...args), wait);
  };
  debounced.cancel = () => {
    window.clearTimeout(timeout);
    timeout = undefined;
  };
  return debounced;
}

function setCursorPosition(input, start, end = start, afterSet) {
  if (!("selectionStart" in input)) {
    afterSet?.();
    return;
  }

  window.setTimeout(() => {
    input.selectionStart = Math.min(start, input.value.length);
    input.selectionEnd = Math.min(end, input.value.length);
    afterSet?.();
  }, 1);
}

function getCursorPosition(input) {
  if ("selectionStart" in input) {
    return {
      start: input.selectionStart,
      end: input.selectionEnd,
    };
  }

  return { start: 0, end: 0 };
}

function setRemoteCursor(clientId, position) {
  remoteCursorOverlay.update(clientId, position);
}

function setSaveState(state) {
  saveStatus.dataset.state = state;

  if (state === "saving") {
    saveStatusText.textContent = "Saving";
    return;
  }

  if (state === "saved") {
    saveStatusText.textContent = "Saved";
  }
}

const markSaveActivity = createSaveActivityIndicator(saveStatus);

function setActionMenuOpen(open, restoreFocus = false) {
  saveMenu.hidden = !open;
  saveActions.classList.toggle("is-open", open);
  saveStatus.setAttribute("aria-expanded", String(open));

  if (open) {
    window.setTimeout(() => copyTextAction.focus(), 0);
  } else {
    if (restoreFocus) {
      saveStatus.focus();
    }
  }
}

function updatePageState() {
  const locked = pageLocked;
  const published = pagePublished;
  const selfDestruct = pageSelfDestruct;
  const encrypted = hasEncryptedBlocks(textarea.value);
  const operationPending = pendingPageOperation !== undefined;

  textarea.readOnly = locked;
  textarea.setAttribute("aria-readonly", String(locked));
  editor.classList.toggle("is-locked", locked);
  saveActions.classList.toggle("is-locked", locked);

  cryptoLockIcon.toggleAttribute("hidden", encrypted);
  cryptoUnlockIcon.toggleAttribute("hidden", !encrypted);
  const cryptoLabel =
    locked && !encrypted
      ? "Unlock page to encrypt paste"
      : encrypted
        ? "Decrypt paste"
        : "Encrypt paste";
  cryptoAction.setAttribute("aria-label", cryptoLabel);
  cryptoAction.dataset.tooltip = cryptoLabel;
  cryptoAction.disabled = (locked && !encrypted) || operationPending;

  pageLockIcon.toggleAttribute("hidden", locked);
  pageUnlockIcon.toggleAttribute("hidden", !locked);
  pageLockAction.setAttribute("aria-label", locked ? "Unlock page" : "Lock page");
  pageLockAction.dataset.tooltip = locked ? "Unlock page" : "Lock page";
  pageLockAction.disabled = operationPending;

  publishIcon.toggleAttribute("hidden", published);
  unpublishIcon.toggleAttribute("hidden", !published);
  publishAction.classList.toggle("is-active", published);
  publishAction.disabled = locked || selfDestruct || operationPending;
  publishAction.setAttribute("aria-checked", String(published));
  publishAction.setAttribute(
    "aria-label",
    locked
      ? "Unlock page to change publishing"
      : selfDestruct
        ? "Cancel self destruct before publishing"
        : published
          ? "Unpublish page"
          : "Publish page",
  );
  publishAction.dataset.tooltip = locked
    ? "Unlock page to change publishing"
    : selfDestruct
      ? "Cancel self destruct before publishing"
      : published
        ? "Unpublish page"
        : "Publish page";

  selfDestructAction.classList.toggle("is-active", selfDestruct);
  selfDestructAction.disabled = locked || operationPending;
  selfDestructAction.setAttribute("aria-checked", String(selfDestruct));
  const selfDestructLabel = locked
    ? "Unlock page to change self destruct"
    : selfDestruct
      ? "Cancel self destruct"
      : "Self destruct after next load";
  selfDestructAction.setAttribute("aria-label", selfDestructLabel);
  selfDestructAction.dataset.tooltip = selfDestructLabel;
}

function replacePageText(
  text,
  locked,
  published,
  selfDestruct,
  restoreCursor = true,
) {
  const cursor = getCursorPosition(textarea);
  cursorBroadcastGuard.pause();
  remoteCursorOverlay.beginTextChange();
  textarea.value = text;
  remoteCursorOverlay.finishTextChange();
  pageLocked = locked;
  pagePublished = published;
  pageSelfDestruct = selfDestruct;
  renderLinks(textarea, linkOverlay);
  updatePageState();
  if (restoreCursor) {
    setCursorPosition(
      textarea,
      cursor.start,
      cursor.end,
      () => cursorBroadcastGuard.resumeAfterCurrentTask(),
    );
  } else {
    cursorBroadcastGuard.resumeAfterCurrentTask();
  }
}

function copyTextFallback(value) {
  const copyTarget = document.createElement("textarea");
  copyTarget.value = value;
  copyTarget.style.position = "fixed";
  copyTarget.style.opacity = "0";
  document.body.append(copyTarget);
  copyTarget.select();
  const copied = document.execCommand("copy");
  copyTarget.remove();

  if (!copied) {
    throw new Error("Copy failed");
  }
}

async function copyPasteText() {
  resetCopyFeedback();

  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(textarea.value);
    } else {
      copyTextFallback(textarea.value);
    }

    copyIcon.hidden = true;
    copySuccessIcon.hidden = false;
    copyTextAction.classList.add("is-success");
    copyTextAction.setAttribute("aria-label", "Paste text copied");
    copyTextAction.dataset.tooltip = "Paste text copied";
    saveStatusText.textContent = "Paste text copied";
    copyFeedbackTimer = window.setTimeout(
      resetCopyFeedback,
      COPY_FEEDBACK_DELAY,
    );
  } catch {
    saveStatusText.textContent = "Could not copy paste text";
  }
}

function setDialogBusy(busy) {
  dialogBusy = busy;
  if (cryptoPassword) {
    cryptoPassword.disabled = busy;
  }
  if (cryptoPasswordConfirmation) {
    cryptoPasswordConfirmation.disabled = busy;
  }
  for (const toggle of cryptoPasswordField.querySelectorAll(
    ".crypto-password-toggle",
  )) {
    toggle.disabled = busy;
  }
  cryptoCancel.disabled = busy;
  cryptoSubmit.disabled = busy;
  cryptoSubmit.textContent = busy
    ? dialogDetails[dialogMode].busy
    : dialogDetails[dialogMode].submit;
}

function createPasswordInput(id, minLength) {
  const passwordInput = document.createElement("input");
  passwordInput.id = id;
  passwordInput.type = "password";
  passwordInput.minLength = minLength;
  passwordInput.required = true;
  passwordInput.autocomplete = "off";
  passwordInput.setAttribute("data-1p-ignore", "");
  passwordInput.setAttribute("data-op-ignore", "");
  return passwordInput;
}

function createPasswordField(id, minLength) {
  const passwordInput = createPasswordInput(id, minLength);
  const passwordControl = document.createElement("div");
  passwordControl.className = "crypto-password-control";

  const visibilityToggle = document.createElement("button");
  visibilityToggle.type = "button";
  visibilityToggle.className = "crypto-password-toggle";
  visibilityToggle.setAttribute("aria-controls", id);

  const closedEyeIcon = createElement(EyeClosed, {
    ...ICON_ATTRIBUTES,
    class: "lucide lucide-eye-closed",
  });
  const openEyeIcon = createElement(Eye, {
    ...ICON_ATTRIBUTES,
    class: "lucide lucide-eye",
  });

  function setPasswordRevealed(revealed) {
    const state = passwordVisibilityState(revealed);
    passwordInput.type = state.inputType;
    visibilityToggle.setAttribute("aria-label", state.label);
    visibilityToggle.setAttribute("aria-pressed", String(revealed));
    visibilityToggle.dataset.tooltip = state.label;
    closedEyeIcon.toggleAttribute("hidden", revealed);
    openEyeIcon.toggleAttribute("hidden", !revealed);
  }

  visibilityToggle.append(closedEyeIcon, openEyeIcon);
  visibilityToggle.addEventListener("click", () => {
    setPasswordRevealed(passwordInput.type === "password");
  });
  setPasswordRevealed(false);

  passwordControl.append(passwordInput, visibilityToggle);
  return { input: passwordInput, control: passwordControl };
}

function mountCryptoPassword(minLength, confirm) {
  const passwordField = createPasswordField("cryptoPassword", minLength);
  const fields = [passwordField.control];

  if (confirm) {
    const confirmationLabel = document.createElement("label");
    confirmationLabel.htmlFor = "cryptoPasswordConfirmation";
    confirmationLabel.textContent = "Confirm password";
    const confirmationField = createPasswordField(
      "cryptoPasswordConfirmation",
      minLength,
    );
    cryptoPasswordConfirmation = confirmationField.input;
    fields.push(confirmationLabel, confirmationField.control);
  } else {
    cryptoPasswordConfirmation = undefined;
  }

  cryptoPasswordField.replaceChildren(...fields);
  cryptoPassword = passwordField.input;
}

function syncPasswordDialogPosition() {
  positionDialogInVisualViewport(cryptoDialog, window.visualViewport);
}

function openPasswordDialog(mode) {
  setActionMenuOpen(false);
  dialogMode = mode;
  const details = dialogDetails[mode];

  cryptoSubmit.textContent = details.submit;
  cryptoDialogDescription.textContent = details.description;
  cryptoDialogDescription.hidden = !details.description;
  mountCryptoPassword(details.minLength, details.confirm);
  cryptoError.textContent = "";
  setDialogBusy(false);
  cryptoDialog.showModal();
  syncPasswordDialogPosition();
  window.setTimeout(() => {
    cryptoPassword?.focus();
    syncPasswordDialogPosition();
  }, 0);
}

function closePasswordDialog() {
  if (cryptoDialog.open) {
    cryptoDialog.close();
  }
}

function sendPageOperation(operation, text, password = "") {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    throw new Error("The server is not connected. Please try again.");
  }

  queueUpdateDebounced.cancel();
  pendingUpdate = undefined;
  const cursor = getCursorPosition(textarea);
  socket.send(
    JSON.stringify(operationMessage(operation, text, password, cursor)),
  );
  lastAnnouncedCursor = cursor;
  pendingPageOperation = operation;
}

async function submitPasswordForm(event) {
  event.preventDefault();
  cryptoError.textContent = "";

  if (!cryptoPassword) {
    return;
  }

  const password = cryptoPassword.value;
  const details = dialogDetails[dialogMode];
  const validationError = validatePasswordFields({
    password,
    confirmation: cryptoPasswordConfirmation?.value,
    minLength: details.minLength,
  });
  if (validationError) {
    cryptoError.textContent = validationError.message;
    if (
      validationError.field === "confirmation" &&
      cryptoPasswordConfirmation
    ) {
      cryptoPasswordConfirmation.focus();
    } else {
      cryptoPassword.focus();
    }
    return;
  }

  if (dialogMode === "encrypt" && pageLocked) {
    cryptoError.textContent = "Unlock the page before encrypting it.";
    cryptoPassword.focus();
    return;
  }

  const sourceText = textarea.value;
  setDialogBusy(true);

  try {
    let transformed = sourceText;
    if (dialogMode === "encrypt" || dialogMode === "decrypt") {
      transformed =
        dialogMode === "encrypt"
          ? await encryptText(sourceText, password)
          : (await decryptEncryptedBlocks(sourceText, password)).text;

      if (textarea.value !== sourceText) {
        throw new Error("The paste changed while processing. Please try again.");
      }
    }

    const serverPassword =
      dialogMode === "lock" || dialogMode === "unlock" ? password : "";
    sendPageOperation(dialogMode, transformed, serverPassword);
    cryptoPassword.value = "";
    if (cryptoPasswordConfirmation) {
      cryptoPasswordConfirmation.value = "";
    }
  } catch (error) {
    cryptoError.textContent =
      error instanceof Error ? error.message : "The operation failed.";
    setDialogBusy(false);
    cryptoPassword.focus();
  }
}

function flashRemoteUpdate() {
  window.clearTimeout(remoteStatusTimer);
  remoteUpdateStatus.classList.add("is-visible");
  remoteStatusTimer = window.setTimeout(() => {
    remoteUpdateStatus.classList.remove("is-visible");
  }, 1000);
}

function flushPendingUpdate() {
  if (!pendingUpdate || !socket || socket.readyState !== WebSocket.OPEN) {
    return;
  }

  socket.send(JSON.stringify(pendingUpdate));
  lastAnnouncedCursor = {
    start: pendingUpdate.cursor_start,
    end: pendingUpdate.cursor_end,
  };
  pendingUpdate = undefined;
  outstandingSaves++;
}

function queueUpdate() {
  if (pageLocked) {
    return;
  }

  const cursor = getCursorPosition(textarea);
  pendingUpdate = editMessage(textarea.value, cursor);
  setSaveState("saving");
  flushPendingUpdate();
}

const queueUpdateDebounced = debounce(queueUpdate, 100);

function sendCursorUpdate() {
  if (
    !cursorBroadcastGuard.canBroadcast() ||
    !socket ||
    socket.readyState !== WebSocket.OPEN
  ) {
    return;
  }

  const cursor = getCursorPosition(textarea);
  if (!cursorPositionChanged(lastAnnouncedCursor, cursor)) {
    return;
  }
  socket.send(JSON.stringify(cursorMessage(cursor)));
  lastAnnouncedCursor = cursor;
}

const queueCursorUpdateDebounced = debounce(sendCursorUpdate, 50);

function finishPageOperation(data) {
  if (data.operation !== pendingPageOperation) {
    return;
  }

  pendingPageOperation = undefined;
  if (
    data.operation === "publish" ||
    data.operation === "unpublish" ||
    data.operation === "self-destruct" ||
    data.operation === "cancel-self-destruct"
  ) {
    pageLocked = data.locked;
    pagePublished = data.published;
    pageSelfDestruct = data.self_destruct;
    updatePageState();
  } else {
    replacePageText(
      data.text,
      data.locked,
      data.published,
      data.self_destruct,
      false,
    );
  }
  if (cryptoDialog.open) {
    setDialogBusy(false);
    closePasswordDialog();
  }
  if (outstandingSaves === 0 && !pendingUpdate) {
    setSaveState("saved");
  }
  if (data.operation === "publish") {
    saveStatusText.textContent = "Published";
  } else if (data.operation === "unpublish") {
    saveStatusText.textContent = "Unpublished";
  } else if (data.operation === "self-destruct") {
    saveStatusText.textContent = "Self destruct armed";
  } else if (data.operation === "cancel-self-destruct") {
    saveStatusText.textContent = "Self destruct cancelled";
  }
}

function handleSocketMessage(event) {
  const data = JSON.parse(event.data);

  if (data.type === websocketMessageType.cursor) {
    setRemoteCursor(data.client_id, data.cursor_end);
    return;
  }

  if (data.type === websocketMessageType.cursorLeave) {
    remoteCursorOverlay.remove(data.client_id);
    return;
  }

  if (data.type === websocketMessageType.ack) {
    if (data.operation) {
      finishPageOperation(data);
      return;
    }

    outstandingSaves = Math.max(0, outstandingSaves - 1);
    if (outstandingSaves === 0 && !pendingUpdate) {
      setSaveState("saved");
    }
    return;
  }

  if (data.type === websocketMessageType.error) {
    if (data.current) {
      replacePageText(
        data.text,
        data.locked,
        data.published,
        data.self_destruct,
      );
    }

    if (data.operation && data.operation === pendingPageOperation) {
      pendingPageOperation = undefined;
      updatePageState();
      if (cryptoDialog.open) {
        cryptoError.textContent = data.error || "The operation failed.";
        setDialogBusy(false);
        cryptoPassword?.focus();
      } else {
        saveStatusText.textContent = data.error || "The operation failed.";
        saveStatus.dataset.state = "saved";
      }
      return;
    }

    outstandingSaves = Math.max(0, outstandingSaves - 1);
    saveStatusText.textContent = data.error || "The paste was not saved.";
    saveStatus.dataset.state = "saved";
    return;
  }

  if (data.type === websocketMessageType.update) {
    replacePageText(
      data.text,
      data.locked,
      data.published,
      data.self_destruct,
    );
    setRemoteCursor(data.client_id, data.cursor_end);
    flashRemoteUpdate();
  }
}

function connectSocket() {
  const place = window.location.pathname.replace(/^\/+/, "");
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  socket = new WebSocket(
    `${protocol}//${window.location.host}/ws?place=${encodeURIComponent(place)}`,
  );

  socket.addEventListener("open", () => {
    flushPendingUpdate();
    sendCursorUpdate();
  });
  socket.addEventListener("message", handleSocketMessage);
  socket.addEventListener("error", () => socket.close());
  socket.addEventListener("close", () => {
    socket = undefined;
    lastAnnouncedCursor = undefined;
    remoteCursorOverlay.clear();

    if (pendingPageOperation) {
      pendingPageOperation = undefined;
      updatePageState();
      if (cryptoDialog.open) {
        cryptoError.textContent =
          "The server disconnected. Enter the password and try again.";
        setDialogBusy(false);
      } else {
        saveStatusText.textContent =
          "The server disconnected. Try the operation again.";
        saveStatus.dataset.state = "saved";
      }
    }

    if (outstandingSaves > 0) {
      outstandingSaves = 0;
      queueUpdate();
    }

    window.clearTimeout(reconnectTimer);
    reconnectTimer = window.setTimeout(connectSocket, 750);
  });
}

textarea.addEventListener("keydown", (event) => {
  if (event.key !== "Tab" || textarea.readOnly) {
    return;
  }

  event.preventDefault();
  const start = textarea.selectionStart;
  const end = textarea.selectionEnd;
  textarea.setRangeText("\t", start, end, "end");
  textarea.dispatchEvent(new Event("input", { bubbles: true }));
});

textarea.addEventListener("beforeinput", () => {
  if (!textarea.readOnly) {
    remoteCursorOverlay.beginTextChange();
  }
});

textarea.addEventListener("input", () => {
  if (textarea.readOnly) {
    return;
  }
  remoteCursorOverlay.finishTextChange();
  markSaveActivity();
  renderLinks(textarea, linkOverlay);
  updatePageState();
  queueUpdateDebounced();
});

textarea.addEventListener("scroll", () => {
  linkOverlay.scrollTop = textarea.scrollTop;
  linkOverlay.scrollLeft = textarea.scrollLeft;
  remoteCursorOverlay.syncScroll();
});

textarea.addEventListener("focus", sendCursorUpdate);

document.addEventListener("selectionchange", () => {
  if (
    document.activeElement === textarea &&
    cursorBroadcastGuard.canBroadcast()
  ) {
    queueCursorUpdateDebounced();
  }
});

saveStatus.addEventListener("click", () => {
  setActionMenuOpen(saveMenu.hidden);
});

copyTextAction.addEventListener("click", () => {
  void copyPasteText();
});

themeAction.addEventListener("click", toggleTheme);
colorSchemeQuery.addEventListener("change", () => {
  if (!selectedTheme) {
    syncTheme();
  }
});

cryptoAction.addEventListener("click", () => {
  const encrypted = hasEncryptedBlocks(textarea.value);
  if (pageLocked && !encrypted) {
    return;
  }
  openPasswordDialog(encrypted ? "decrypt" : "encrypt");
});
pageLockAction.addEventListener("click", () => {
  openPasswordDialog(pageLocked ? "unlock" : "lock");
});
publishAction.addEventListener("click", () => {
  if (pageLocked || pageSelfDestruct) {
    return;
  }

  try {
    setSaveState("saving");
    queueUpdateDebounced.cancel();
    queueUpdate();
    sendPageOperation(
      pagePublished ? "unpublish" : "publish",
      "",
    );
    updatePageState();
  } catch (error) {
    saveStatusText.textContent =
      error instanceof Error ? error.message : "The operation failed.";
    saveStatus.dataset.state = "saved";
  }
});
selfDestructAction.addEventListener("click", () => {
  if (pageLocked) {
    return;
  }

  try {
    setSaveState("saving");
    queueUpdateDebounced.cancel();
    queueUpdate();
    sendPageOperation(
      pageSelfDestruct ? "cancel-self-destruct" : "self-destruct",
      "",
    );
    updatePageState();
  } catch (error) {
    saveStatusText.textContent =
      error instanceof Error ? error.message : "The operation failed.";
    saveStatus.dataset.state = "saved";
  }
});
cryptoForm.addEventListener("submit", (event) => {
  void submitPasswordForm(event);
});
cryptoCancel.addEventListener("click", closePasswordDialog);

cryptoDialog.addEventListener("close", () => {
  cryptoError.textContent = "";
  setDialogBusy(false);
  resetDialogVisualViewportPosition(cryptoDialog);
  cryptoPasswordField.replaceChildren();
  cryptoPassword = undefined;
  cryptoPasswordConfirmation = undefined;
});

cryptoDialog.addEventListener("cancel", (event) => {
  if (dialogBusy) {
    event.preventDefault();
  }
});

cryptoDialog.addEventListener("click", (event) => {
  if (!dialogBusy && event.target === cryptoDialog) {
    closePasswordDialog();
  }
});

window.visualViewport?.addEventListener(
  "resize",
  syncPasswordDialogPosition,
);
window.visualViewport?.addEventListener(
  "scroll",
  syncPasswordDialogPosition,
);

document.addEventListener("pointerdown", (event) => {
  if (!saveMenu.hidden && !saveActions.contains(event.target)) {
    setActionMenuOpen(false);
  }
});

document.addEventListener("keydown", (event) => {
  if (event.key === "Escape" && !saveMenu.hidden) {
    event.preventDefault();
    setActionMenuOpen(false, true);
  }
});

renderLinks(textarea, linkOverlay);
syncTheme();
updatePageState();
connectSocket();

setCursorPosition(
  textarea,
  Number(textarea.dataset.cursorStart) || 0,
  Number(textarea.dataset.cursorEnd) || 0,
);
textarea.focus();
