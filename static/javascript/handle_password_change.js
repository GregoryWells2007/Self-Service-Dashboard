document
  .getElementById("close_password_dialogue")
  .addEventListener("click", () => {
    document.getElementById("change_password_dialogue").classList.add("hidden");
    document.getElementById("popup_background").classList.add("hidden");
  });

const popup_botton = document.getElementById("change_password_button");

const currentPasswordButton = document.getElementById("current_password"),
  newPasswordButton = document.getElementById("new_password"),
  newPasswordRepeatButton = document.getElementById("new_password_repeat");

const strengh_label = document.getElementById("strengh-label");
const password_progress = document.getElementById("password-progress");

popup_botton.addEventListener("click", () => {
  document.getElementById("popup_background").classList.remove("hidden");
  document
    .getElementById("change_password_dialogue")
    .classList.remove("hidden");

  currentPasswordButton.value = "";
  newPasswordButton.value = "";
  newPasswordRepeatButton.value = "";
  strengh_label.innerText = "Strength: Weak";
  password_progress.style.width = "0%";
});

const changePasswordButton = document.getElementById(
  "final_change_password_button",
);

function displayError(errorText) {
  document.getElementById("password_error").classList.remove("hidden");
  document.getElementById("password_text").innerText = "⚠️ " + errorText + ".";
  return;
}

changePasswordButton.addEventListener("click", () => {
  if (currentPasswordButton.value === "") {
    displayError("Please enter current password");
    return;
  }

  if (newPasswordButton.value === "") {
    displayError("No value for new password");
    return;
  }

  if (newPasswordRepeatButton.value === "") {
    displayError("Please repeat new password");
    return;
  }

  if (newPasswordButton.value !== newPasswordRepeatButton.value) {
    displayError("New passwords do not match");
    return;
  }

  const formData = new FormData();
  formData.append(
    "csrf_token",
    document.getElementById("csrf_token_storage").value,
  );
  formData.append("old_password", currentPasswordButton.value);
  formData.append("new_password", newPasswordButton.value);
  formData.append("new_password_repeat", newPasswordRepeatButton.value);

  fetch("/change-password", {
    method: "POST",
    body: formData,
  })
    .then(async (res) => {
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || "Request failed");
      }
      return data;
    })
    .then((data) => {
      document
        .getElementById("change_password_dialogue")
        .classList.add("hidden");
      document.getElementById("popup_background").classList.add("hidden");
    })
    .catch((err) => {
      displayError(err.message);
    });
});

document.getElementById("new_password").addEventListener("input", () => {
  score = EvaluatePassword(document.getElementById("new_password").value).score;
  password_progress.style.width = score + "%";

  if (score <= 40) {
    strengh_label.innerText = "Strength: Weak";
    password_progress.style.backgroundColor = "var(--password-strength-weak)";
  } else if (score > 40 && score <= 70) {
    strengh_label.innerText = "Strength: Medium";
    password_progress.style.backgroundColor = "var(--password-strength-medium)";
  } else if (score > 40 && score >= 70) {
    strengh_label.innerText = "Strength: Strong";
    password_progress.style.backgroundColor = "var(--password-strength-strong)";
  }
});
