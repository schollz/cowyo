export function validatePasswordFields({
  password,
  confirmation,
  minLength,
}) {
  if (password.length < minLength) {
    return {
      field: "password",
      message:
        minLength === 8
          ? "Use a password with at least 8 characters."
          : "Enter the password.",
    };
  }

  if (confirmation !== undefined && password !== confirmation) {
    return {
      field: "confirmation",
      message: "Passwords do not match.",
    };
  }

  return undefined;
}

export function passwordVisibilityState(revealed) {
  return revealed
    ? {
        inputType: "text",
        label: "Hide password",
      }
    : {
        inputType: "password",
        label: "Show password",
      };
}
