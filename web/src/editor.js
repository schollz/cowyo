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
  KeyRound,
  LockKeyhole,
  Moon,
  ShieldCheck,
  ShieldKeyhole,
  Sun,
  UnlockKeyhole,
  X,
} from "lucide";

import {
  decryptEncryptedBlocks,
  encryptText,
  hasEncryptedBlocks,
} from "./encryption.js";
import {
  bytesToBase64URL,
  createSerialQueue,
  decryptE2EEDocument,
  derivePageKeys,
  encodeMasterKey,
  encryptE2EEDocument,
  generateMasterKey,
  normalizePagePath,
  parseFragmentKey,
  privatePageURL,
} from "./e2ee.js";
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
import { createPrivateStatusController } from "./private-status.js";
import {
  calculateAnchoredStatusPosition,
  measureTextareaEnd,
} from "./private-status-position.js";
import {
  cursorMessage,
  e2eeAuthenticateMessage,
  e2eeBootstrapMessage,
  editMessage,
  operationMessage,
  websocketURL,
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
const privateAction = document.getElementById("privateAction");
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
const privateConversionDialog = document.getElementById(
  "privateConversionDialog",
);
const privateConversionForm = document.getElementById(
  "privateConversionForm",
);
const privateConversionCancel = document.getElementById(
  "privateConversionCancel",
);
const remoteUpdateStatus = document.getElementById("remoteUpdateStatus");
const privateStatus = document.getElementById("privateStatus");
const privateStatusText = document.getElementById("privateStatusText");
const privateStatusClose = document.getElementById("privateStatusClose");
const privateStatusController = createPrivateStatusController(
  privateStatus,
  privateStatusText,
);

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
    KeyRound,
    LockKeyhole,
    Moon,
    ShieldCheck,
    ShieldKeyhole,
    Sun,
    UnlockKeyhole,
    X,
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
let pageE2EE = textarea.dataset.pageE2ee === "true";
let privateBootstrap = textarea.dataset.privateBootstrap === "true";
let conversionBootstrap = textarea.dataset.conversionBootstrap === "true";
let e2eeAuthenticated = false;
let e2eeFinal = false;
let e2eeKeys;
let encodedMasterKey;
let latestCiphertext = pageE2EE ? textarea.value : "";
let pendingE2EEBootstrap;
let pendingE2EESnapshot;
let e2eeTextRevision = 0;
let e2eeRemoteSequence = 0;
let e2eeDecrypting = false;
let reconnectEnabled = true;
const e2eeCryptoQueue = createSerialQueue();
const ordinarySaveWaiters = new Set();
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
let privateStatusPositionFrame;

function positionPrivateKeyError({ revealEnd = false } = {}) {
  if (
    privateStatus.hidden ||
    privateStatus.dataset.keyError !== "true"
  ) {
    return;
  }
  if (revealEnd) {
    textarea.scrollTop = textarea.scrollHeight;
  }

  const anchor = measureTextareaEnd(textarea);
  const statusRect = privateStatus.getBoundingClientRect();
  const position = calculateAnchoredStatusPosition({
    anchorLeft: anchor.left,
    anchorTop: anchor.top,
    anchorBottom: anchor.bottom,
    statusWidth: statusRect.width,
    statusHeight: statusRect.height,
    viewportWidth: window.innerWidth,
    viewportHeight: window.innerHeight,
  });
  privateStatus.dataset.placement = position.placement;
  privateStatus.style.setProperty(
    "--private-status-left",
    `${position.left}px`,
  );
  privateStatus.style.setProperty(
    "--private-status-pointer-left",
    `${position.pointerLeft}px`,
  );
  privateStatus.style.setProperty(
    "--private-status-top",
    `${position.top}px`,
  );
}

function schedulePrivateKeyErrorPosition(revealEnd = false) {
  window.cancelAnimationFrame(privateStatusPositionFrame);
  privateStatusPositionFrame = window.requestAnimationFrame(() => {
    positionPrivateKeyError({ revealEnd });
  });
}

function showPrivateStatus(message, error = false) {
  privateStatusController.show(message, error);
  if (privateStatus.dataset.keyError === "true") {
    schedulePrivateKeyErrorPosition(true);
  }
}

function settleOrdinarySaveWaiters(error) {
  if (outstandingSaves !== 0 || pendingUpdate || pendingE2EESnapshot) {
    return;
  }
  for (const waiter of ordinarySaveWaiters) {
    window.clearTimeout(waiter.timeout);
    if (error) {
      waiter.reject(error);
    } else {
      waiter.resolve();
    }
  }
  ordinarySaveWaiters.clear();
}

function waitForOrdinarySaves() {
  if (
    outstandingSaves === 0 &&
    !pendingUpdate &&
    !pendingE2EESnapshot
  ) {
    return Promise.resolve();
  }
  return new Promise((resolve, reject) => {
    const waiter = { resolve, reject };
    waiter.timeout = window.setTimeout(() => {
      ordinarySaveWaiters.delete(waiter);
      reject(new Error("The current text could not be confirmed as saved."));
    }, 10000);
    ordinarySaveWaiters.add(waiter);
  });
}

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

  const privateReady = !pageE2EE || (e2eeAuthenticated && !e2eeFinal);
  const readOnly =
    locked ||
    !privateReady ||
    privateBootstrap ||
    conversionBootstrap ||
    e2eeDecrypting;

  textarea.readOnly = readOnly;
  textarea.setAttribute("aria-readonly", String(readOnly));
  editor.classList.toggle("is-locked", readOnly);
  saveActions.classList.toggle("is-locked", readOnly);
  copyTextAction.disabled = pageE2EE && !e2eeAuthenticated;

  cryptoLockIcon.toggleAttribute("hidden", encrypted && !pageE2EE);
  cryptoUnlockIcon.toggleAttribute("hidden", !encrypted || pageE2EE);
  const cryptoLabel =
    pageE2EE
      ? "Legacy block encryption is unavailable for private pages"
      : locked && !encrypted
      ? "Unlock page to encrypt paste"
      : encrypted
        ? "Decrypt paste"
        : "Encrypt paste";
  cryptoAction.setAttribute("aria-label", cryptoLabel);
  cryptoAction.dataset.tooltip = cryptoLabel;
  cryptoAction.disabled =
    pageE2EE || (locked && !encrypted) || operationPending;

  privateAction.classList.toggle("is-active", pageE2EE);
  const privateLabel = pageE2EE
    ? privateBootstrap || conversionBootstrap
      ? "Creating private page…"
      : encodedMasterKey
      ? "Copy complete private URL"
      : "A valid private key is required"
    : locked
      ? "Unlock page before making it private"
      : selfDestruct
        ? "Cancel self destruct before making it private"
        : "Make permanently private";
  privateAction.setAttribute("aria-label", privateLabel);
  privateAction.dataset.tooltip = privateLabel;
  privateAction.disabled = pageE2EE
    ? !encodedMasterKey || privateBootstrap || conversionBootstrap
    : locked || selfDestruct || operationPending;

  pageLockIcon.toggleAttribute("hidden", locked);
  pageUnlockIcon.toggleAttribute("hidden", !locked);
  pageLockAction.setAttribute("aria-label", locked ? "Unlock page" : "Lock page");
  pageLockAction.dataset.tooltip = locked ? "Unlock page" : "Lock page";
  pageLockAction.disabled = operationPending || (pageE2EE && !e2eeAuthenticated);

  publishIcon.toggleAttribute("hidden", published);
  unpublishIcon.toggleAttribute("hidden", !published);
  publishAction.classList.toggle("is-active", published);
  publishAction.disabled =
    pageE2EE || locked || selfDestruct || operationPending;
  publishAction.setAttribute("aria-checked", String(published));
  publishAction.setAttribute(
    "aria-label",
    pageE2EE
      ? "Publishing is unavailable for private pages"
      : locked
      ? "Unlock page to change publishing"
      : selfDestruct
        ? "Cancel self destruct before publishing"
        : published
          ? "Unpublish page"
          : "Publish page",
  );
  publishAction.dataset.tooltip = pageE2EE
    ? "Publishing is unavailable for private pages"
    : locked
    ? "Unlock page to change publishing"
    : selfDestruct
      ? "Cancel self destruct before publishing"
      : published
        ? "Unpublish page"
        : "Publish page";

  selfDestructAction.classList.toggle("is-active", selfDestruct);
  selfDestructAction.disabled =
    locked || operationPending || (pageE2EE && !e2eeAuthenticated);
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
  endToEndEncrypted = pageE2EE,
) {
  const cursor = getCursorPosition(textarea);
  cursorBroadcastGuard.pause();
  remoteCursorOverlay.beginTextChange();
  textarea.value = text;
  remoteCursorOverlay.finishTextChange();
  pageLocked = locked;
  pagePublished = published;
  pageSelfDestruct = selfDestruct;
  pageE2EE = endToEndEncrypted;
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

async function copyValue(value) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(value);
  } else {
    copyTextFallback(value);
  }
}

async function copyPasteText() {
  resetCopyFeedback();

  try {
    await copyValue(textarea.value);

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

async function copyPrivateLink() {
  if (!pageE2EE || !encodedMasterKey) {
    return;
  }
  try {
    await copyValue(privatePageURL(window.location, encodedMasterKey));
    saveStatusText.textContent = "Complete private URL copied";
    showPrivateStatus(
      "Complete private URL copied. Anyone with this URL can read and edit the scratchpad.",
    );
  } catch {
    saveStatusText.textContent = "Could not copy the private URL";
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

function syncPrivateConversionDialogPosition() {
  positionDialogInVisualViewport(
    privateConversionDialog,
    window.visualViewport,
  );
}

function openPrivateConversionDialog() {
  setActionMenuOpen(false);
  privateConversionDialog.showModal();
  syncPrivateConversionDialogPosition();
  window.setTimeout(() => {
    privateConversionCancel.focus();
    syncPrivateConversionDialogPosition();
  }, 0);
}

function closePrivateConversionDialog() {
  if (privateConversionDialog.open) {
    privateConversionDialog.close();
  }
}

async function preparePrivateConversion() {
  if (
    pageE2EE ||
    pageLocked ||
    pageSelfDestruct ||
    pendingPageOperation
  ) {
    return;
  }

  try {
    setSaveState("saving");
    queueUpdateDebounced.cancel();
    queueUpdate();
    await waitForOrdinarySaves();
    const target = new URL(window.location.href);
    target.search = "?convert=1";
    target.hash = "";
    window.location.assign(target.toString());
  } catch (error) {
    saveStatusText.textContent =
      error instanceof Error
        ? error.message
        : "The page could not be prepared for conversion.";
    saveStatus.dataset.state = "saved";
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

async function sendPageOperationAfterSave(operation, text, password = "") {
  if (pageE2EE && !pageLocked) {
    queueUpdateDebounced.cancel();
    queueUpdate();
    await e2eeCryptoQueue.drain();
    await waitForOrdinarySaves();
  }
  sendPageOperation(operation, text, password);
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
    await sendPageOperationAfterSave(dialogMode, transformed, serverPassword);
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

function flushPendingE2EEUpdate() {
  if (!pendingE2EESnapshot || !e2eeAuthenticated || e2eeFinal) {
    return;
  }
  const snapshot = pendingE2EESnapshot;
  if (snapshot.queued) {
    return;
  }
  snapshot.queued = true;
  void e2eeCryptoQueue
    .run(async () => {
      if (
        !e2eeKeys ||
        !socket ||
        socket.readyState !== WebSocket.OPEN ||
        !e2eeAuthenticated ||
        e2eeFinal
      ) {
        snapshot.queued = false;
        return;
      }
      const ciphertext = encryptE2EEDocument(
        snapshot.text,
        e2eeKeys.contentKey,
        window.location.pathname,
      );
      socket.send(JSON.stringify(editMessage(ciphertext, snapshot.cursor)));
      latestCiphertext = ciphertext;
      lastAnnouncedCursor = snapshot.cursor;
      if (pendingE2EESnapshot === snapshot) {
        pendingE2EESnapshot = undefined;
      }
      outstandingSaves++;
    })
    .catch((error) => {
      snapshot.queued = false;
      saveStatusText.textContent =
        error instanceof Error ? error.message : "The private page was not saved.";
      saveStatus.dataset.state = "saved";
    });
}

function queueUpdate() {
  if (pageLocked || e2eeFinal) {
    return;
  }

  const cursor = getCursorPosition(textarea);
  if (pageE2EE) {
    if (!e2eeAuthenticated) {
      return;
    }
    pendingE2EESnapshot = { text: textarea.value, cursor };
    setSaveState("saving");
    flushPendingE2EEUpdate();
    return;
  }
  pendingUpdate = editMessage(textarea.value, cursor);
  setSaveState("saving");
  flushPendingUpdate();
}

const queueUpdateDebounced = debounce(() => {
  if (pageE2EE) {
    flushPendingE2EEUpdate();
  } else {
    queueUpdate();
  }
}, 100);

function sendCursorUpdate() {
  if (
    !cursorBroadcastGuard.canBroadcast() ||
    !socket ||
    socket.readyState !== WebSocket.OPEN ||
    (pageE2EE && !e2eeAuthenticated)
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
  if (pageE2EE) {
    latestCiphertext = data.text || latestCiphertext;
    pageLocked = data.locked;
    pagePublished = false;
    pageSelfDestruct = data.self_destruct;
    updatePageState();
  } else if (
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

function revealRawPrivateDocument(data, message) {
  reconnectEnabled = false;
  e2eeAuthenticated = false;
  e2eeDecrypting = false;
  pageE2EE = true;
  latestCiphertext = data.text || latestCiphertext;
  replacePageText(
    latestCiphertext,
    data.locked,
    false,
    data.self_destruct,
    false,
    true,
  );
  showPrivateStatus(message, true);
  socket?.close();
}

function applyRemoteE2EEUpdate(
  data,
  { final = false, showActiveStatus = false } = {},
) {
  const sequence = ++e2eeRemoteSequence;
  const revision = e2eeTextRevision;
  latestCiphertext = data.text;
  e2eeDecrypting = true;
  updatePageState();

  void e2eeCryptoQueue
    .run(() =>
      decryptE2EEDocument(
        data.text,
        e2eeKeys.contentKey,
        window.location.pathname,
      ),
    )
    .then((plaintext) => {
      if (sequence !== e2eeRemoteSequence || revision !== e2eeTextRevision) {
        return;
      }
      e2eeDecrypting = false;
      e2eeFinal = final;
      replacePageText(
        plaintext,
        data.locked,
        false,
        data.self_destruct,
        !final,
        true,
      );
      if (final) {
        reconnectEnabled = false;
        showPrivateStatus(
          "This was the page’s final authorized load. It has been deleted from the server.",
        );
        socket?.close();
      } else {
        if (showActiveStatus) {
          privateStatusController.showActiveWhenEmpty(plaintext);
        } else {
          privateStatusController.hideActiveWhenContent(plaintext);
        }
        flushPendingE2EEUpdate();
      }
    })
    .catch((error) => {
      revealRawPrivateDocument(
        data,
        error instanceof Error
          ? error.message
          : "The private document could not be decrypted.",
      );
    });
}

function finishE2EEAuthentication(data) {
  pageE2EE = true;
  e2eeAuthenticated = true;
  pageLocked = data.locked;
  pagePublished = false;
  pageSelfDestruct = data.self_destruct;
  latestCiphertext = data.text;

  if (
    data.operation === websocketMessageType.e2eeCreate ||
    data.operation === websocketMessageType.e2eeConvert ||
    pendingE2EEBootstrap
  ) {
    pendingE2EEBootstrap = undefined;
    privateBootstrap = false;
    conversionBootstrap = false;
    window.history.replaceState(
      null,
      "",
      privatePageURL(window.location, encodedMasterKey),
    );
    updatePageState();
    privateStatusController.showActiveWhenEmpty(textarea.value);
    setSaveState("saved");
    return;
  }

  applyRemoteE2EEUpdate(data, {
    final: data.final === true,
    showActiveStatus: true,
  });
}

function handleSocketMessage(event) {
  const data = JSON.parse(event.data);

  if (data.type === websocketMessageType.e2eeAuthenticated) {
    finishE2EEAuthentication(data);
    return;
  }

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
    if (
      outstandingSaves === 0 &&
      !pendingUpdate &&
      !pendingE2EESnapshot
    ) {
      setSaveState("saved");
    }
    settleOrdinarySaveWaiters();
    return;
  }

  if (data.type === websocketMessageType.error) {
    if (data.operation === websocketMessageType.e2eeCreate) {
      if (data.end_to_end_encrypted && e2eeKeys) {
        socket.send(
          JSON.stringify(
            e2eeAuthenticateMessage(bytesToBase64URL(e2eeKeys.writeKey)),
          ),
        );
        return;
      }
      if (data.error_code !== "page-name-collision") {
        reconnectEnabled = false;
        showPrivateStatus(
          data.error || "The private page could not be created.",
          true,
        );
        socket?.close();
        return;
      }
      reconnectEnabled = false;
      showPrivateStatus(
        "That random name was already in use. Allocating another private page…",
        true,
      );
      socket?.close();
      window.location.replace("/?new=private");
      return;
    }
    if (
      data.operation === websocketMessageType.e2eeConvert ||
      data.operation === websocketMessageType.e2eeAuthenticate
    ) {
      if (
        data.operation === websocketMessageType.e2eeConvert &&
        data.end_to_end_encrypted &&
        e2eeKeys
      ) {
        socket.send(
          JSON.stringify(
            e2eeAuthenticateMessage(bytesToBase64URL(e2eeKeys.writeKey)),
          ),
        );
        return;
      }
      if (
        data.operation === websocketMessageType.e2eeConvert &&
        !data.end_to_end_encrypted
      ) {
        reconnectEnabled = false;
        window.location.replace(window.location.pathname);
        return;
      }
      if (
        data.operation === websocketMessageType.e2eeAuthenticate &&
        pendingE2EEBootstrap?.type === websocketMessageType.e2eeCreate
      ) {
        reconnectEnabled = false;
        window.location.replace("/?new=private");
        return;
      }
      if (
        data.operation === websocketMessageType.e2eeAuthenticate &&
        pendingE2EEBootstrap?.type === websocketMessageType.e2eeConvert
      ) {
        reconnectEnabled = false;
        window.location.replace(window.location.pathname);
        return;
      }
      revealRawPrivateDocument(
        data,
        data.error || "The private page could not be authenticated.",
      );
      return;
    }
    if (data.current && !(pageE2EE && e2eeAuthenticated)) {
      if (data.end_to_end_encrypted) {
        revealRawPrivateDocument(
          data,
          data.error || "A valid #key private URL is required to edit this page.",
        );
      } else {
        replacePageText(
          data.text,
          data.locked,
          data.published,
          data.self_destruct,
        );
      }
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
    settleOrdinarySaveWaiters(new Error(data.error || "The paste was not saved."));
    return;
  }

  if (data.type === websocketMessageType.update) {
    if (data.end_to_end_encrypted) {
      if (!e2eeKeys || !e2eeAuthenticated) {
        revealRawPrivateDocument(
          data,
          "This page was converted to permanent end-to-end encryption. Ask the converter for the new complete #key URL.",
        );
        return;
      }
      if (!data.operation && pendingE2EESnapshot) {
        latestCiphertext = data.text;
        if (data.client_id) {
          setRemoteCursor(data.client_id, data.cursor_end);
        }
        flashRemoteUpdate();
        return;
      }
      applyRemoteE2EEUpdate(data);
      if (data.client_id) {
        setRemoteCursor(data.client_id, data.cursor_end);
      }
      flashRemoteUpdate();
      return;
    }
    replacePageText(
      data.text,
      data.locked,
      data.published,
      data.self_destruct,
    );
    if (data.client_id) {
      setRemoteCursor(data.client_id, data.cursor_end);
    }
    flashRemoteUpdate();
  }
}

function connectSocket() {
  const place = normalizePagePath(window.location.pathname).slice(1);
  socket = new WebSocket(websocketURL(window.location, place));

  socket.addEventListener("open", () => {
    if (pendingE2EEBootstrap) {
      socket.send(JSON.stringify(pendingE2EEBootstrap));
      return;
    }
    if (pageE2EE && e2eeKeys) {
      socket.send(
        JSON.stringify(
          e2eeAuthenticateMessage(bytesToBase64URL(e2eeKeys.writeKey)),
        ),
      );
      return;
    }
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

    if (pageE2EE) {
      if (outstandingSaves > 0 && !pageLocked && !e2eeFinal) {
        pendingE2EESnapshot = {
          text: textarea.value,
          cursor: getCursorPosition(textarea),
        };
      }
      outstandingSaves = 0;
      e2eeAuthenticated = false;
      if (!e2eeFinal) {
        updatePageState();
      }
    } else if (outstandingSaves > 0) {
      outstandingSaves = 0;
      queueUpdate();
    }

    window.clearTimeout(reconnectTimer);
    if (reconnectEnabled && !e2eeFinal) {
      reconnectTimer = window.setTimeout(connectSocket, 750);
    }
  });
}

async function initializePrivateBootstrap(type) {
  pageE2EE = true;
  e2eeAuthenticated = false;
  showPrivateStatus("Creating the private page entirely in this browser…");
  updatePageState();

  try {
    const masterKey = generateMasterKey();
    encodedMasterKey = encodeMasterKey(masterKey);
    e2eeKeys = derivePageKeys(masterKey, window.location.pathname);
    masterKey.fill(0);
    const ciphertext = encryptE2EEDocument(
      textarea.value,
      e2eeKeys.contentKey,
      window.location.pathname,
    );
    latestCiphertext = ciphertext;
    const cursor = getCursorPosition(textarea);
    pendingE2EEBootstrap = e2eeBootstrapMessage(
      type,
      ciphertext,
      bytesToBase64URL(e2eeKeys.writeKey),
      cursor,
    );
    connectSocket();
  } catch (error) {
    reconnectEnabled = false;
    showPrivateStatus(
      error instanceof Error
        ? error.message
        : "The private page could not be created.",
      true,
    );
  }
}

async function initializeExistingE2EE() {
  const rawCiphertext = textarea.value;
  pagePublished = false;
  updatePageState();

  try {
    const masterKey = parseFragmentKey(window.location.hash);
    encodedMasterKey = encodeMasterKey(masterKey);
    e2eeKeys = derivePageKeys(masterKey, window.location.pathname);
    masterKey.fill(0);
    if (!pageSelfDestruct) {
      const plaintext = await e2eeCryptoQueue.run(() =>
        decryptE2EEDocument(
          rawCiphertext,
          e2eeKeys.contentKey,
          window.location.pathname,
        ),
      );
      replacePageText(
        plaintext,
        pageLocked,
        false,
        false,
        false,
        true,
      );
    } else {
      showPrivateStatus(
        "Authenticating before retrieving this self-destructing private page…",
      );
    }
    connectSocket();
  } catch (error) {
    reconnectEnabled = false;
    e2eeKeys = undefined;
    encodedMasterKey = undefined;
    replacePageText(
      rawCiphertext,
      pageLocked,
      false,
      pageSelfDestruct,
      false,
      true,
    );
    showPrivateStatus(
      error instanceof Error
        ? error.message
        : "A valid #key private URL is required.",
      true,
    );
  }
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
  if (pageE2EE) {
    e2eeTextRevision++;
    privateStatusController.hideActiveWhenContent(textarea.value);
  }
  markSaveActivity();
  renderLinks(textarea, linkOverlay);
  updatePageState();
  if (pageE2EE) {
    pendingE2EESnapshot = {
      text: textarea.value,
      cursor: getCursorPosition(textarea),
    };
    setSaveState("saving");
  }
  queueUpdateDebounced();
});

textarea.addEventListener("scroll", () => {
  linkOverlay.scrollTop = textarea.scrollTop;
  linkOverlay.scrollLeft = textarea.scrollLeft;
  remoteCursorOverlay.syncScroll();
  schedulePrivateKeyErrorPosition();
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
  if (pageE2EE) {
    return;
  }
  const encrypted = hasEncryptedBlocks(textarea.value);
  if (pageLocked && !encrypted) {
    return;
  }
  openPasswordDialog(encrypted ? "decrypt" : "encrypt");
});
privateAction.addEventListener("click", () => {
  if (pageE2EE) {
    void copyPrivateLink();
    return;
  }
  if (pageLocked || pageSelfDestruct || pendingPageOperation) {
    return;
  }
  openPrivateConversionDialog();
});
pageLockAction.addEventListener("click", () => {
  openPasswordDialog(pageLocked ? "unlock" : "lock");
});
publishAction.addEventListener("click", () => {
  if (pageE2EE || pageLocked || pageSelfDestruct) {
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

  void (async () => {
    try {
      setSaveState("saving");
      queueUpdateDebounced.cancel();
      if (!pageE2EE) {
        queueUpdate();
      }
      await sendPageOperationAfterSave(
        pageSelfDestruct ? "cancel-self-destruct" : "self-destruct",
        "",
      );
      updatePageState();
    } catch (error) {
      saveStatusText.textContent =
        error instanceof Error ? error.message : "The operation failed.";
      saveStatus.dataset.state = "saved";
    }
  })();
});
cryptoForm.addEventListener("submit", (event) => {
  void submitPasswordForm(event);
});
cryptoCancel.addEventListener("click", closePasswordDialog);
privateConversionForm.addEventListener("submit", (event) => {
  event.preventDefault();
  closePrivateConversionDialog();
  void preparePrivateConversion();
});
privateConversionCancel.addEventListener(
  "click",
  closePrivateConversionDialog,
);

privateStatusClose.addEventListener("click", () => {
  privateStatusController.dismiss();
  textarea.focus({ preventScroll: true });
});

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

privateConversionDialog.addEventListener("close", () => {
  resetDialogVisualViewportPosition(privateConversionDialog);
});

privateConversionDialog.addEventListener("click", (event) => {
  if (event.target === privateConversionDialog) {
    closePrivateConversionDialog();
  }
});

window.visualViewport?.addEventListener(
  "resize",
  syncPasswordDialogPosition,
);
window.visualViewport?.addEventListener(
  "resize",
  syncPrivateConversionDialogPosition,
);
window.visualViewport?.addEventListener(
  "scroll",
  syncPasswordDialogPosition,
);
window.visualViewport?.addEventListener(
  "scroll",
  syncPrivateConversionDialogPosition,
);

document.addEventListener("pointerdown", (event) => {
  privateStatusController.dismissWhenOutside(event.target);
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

window.addEventListener("resize", () => {
  schedulePrivateKeyErrorPosition(true);
});
void document.fonts?.ready.then(() => {
  schedulePrivateKeyErrorPosition(true);
});

renderLinks(textarea, linkOverlay);
syncTheme();
setCursorPosition(
  textarea,
  Number(textarea.dataset.cursorStart) || 0,
  Number(textarea.dataset.cursorEnd) || 0,
);
updatePageState();

if (privateBootstrap) {
  void initializePrivateBootstrap(websocketMessageType.e2eeCreate);
} else if (conversionBootstrap) {
  void initializePrivateBootstrap(websocketMessageType.e2eeConvert);
} else if (pageE2EE) {
  void initializeExistingE2EE();
} else {
  connectSocket();
}

if (!textarea.readOnly) {
  textarea.focus();
}
