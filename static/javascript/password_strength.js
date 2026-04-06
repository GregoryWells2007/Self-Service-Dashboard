// build in evaluate password function
// Errors: reasons why the FreeIPA server would reject the password
// Suggestions: reasons the password should be made stronger
// You can change this code to change how complexity is rated
// Return values
// Score: 0-100
// Errors: A list of errors that would cause the server to reject the password
// Suggestions: A list of suggestions to make the password stronger
function EvaluatePassword(password) {
  let score = 0;
  let errors = [];
  let suggestions = [];

  const hasUpper = /[A-Z]/.test(password);
  const hasLower = /[a-z]/.test(password);
  const hasNumber = /[0-9]/.test(password);
  const hasSpecial = /[^A-Za-z0-9]/.test(password);
  const isLongEnough = password.length >= 8;

  if (!isLongEnough) {
    errors.push("Password must be at least 8 characters long.");
  }
  score += Math.min(password.length * 3, 60);
  if (hasUpper) score += 10;
  if (hasLower) score += 10;
  if (hasNumber) score += 10;
  if (hasSpecial) score += 10;
  if (score < 100) {
    if (password.length < 20) {
      suggestions.push(
        `Add ${20 - password.length} more characters to reach maximum length points.`,
      );
    }
    if (!hasUpper) suggestions.push("Add an uppercase letter.");
    if (!hasLower) suggestions.push("Add a lowercase letter.");
    if (!hasNumber) suggestions.push("Add a number.");
    if (!hasSpecial)
      suggestions.push("Add a special character (e.g., !, @, #).");
    if (score > 70 && score < 100) {
      suggestions.push(
        "Pro-tip: Use a full sentence (passphrase) to make it easier to remember and harder to crack.",
      );
    }
  }

  if (!accepted && isLongEnough) {
    errors.push("Password is too simple. Try adding more variety or length.");
  }
  return {
    score: Math.min(score, 100),
    errors: errors,
    suggestions: suggestions,
  };
}
