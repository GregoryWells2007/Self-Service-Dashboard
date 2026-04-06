document
  .getElementById("close_password_dialogue")
  .addEventListener("click", () => {
    document.getElementById("change_password_dialogue").classList.add("hidden");
    document.getElementById("popup_background").classList.add("hidden");
  });

const popup_botton = document.getElementById("change_password_button");

popup_botton.addEventListener("click", () => {
  document.getElementById("popup_background").classList.remove("hidden");
  document
    .getElementById("change_password_dialogue")
    .classList.remove("hidden");
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
  if (document.getElementById("current_password").value === "") {
    displayError("Please enter current password");
    return;
  }

  if (document.getElementById("new_password").value === "") {
    displayError("No value for new password");
    return;
  }

  if (document.getElementById("new_password_repeat").value === "") {
    displayError("Please repeat new password");
    return;
  }

  if (
    document.getElementById("new_password").value !==
    document.getElementById("new_password_repeat").value
  ) {
    displayError("New password and new password repeat do not match");
    return;
  }

  const formData = new FormData();
  formData.append(
    "csrf_token",
    document.getElementById("csrf_token_storage").value,
  );
  formData.append(
    "old_password",
    document.getElementById("current_password").value,
  );
  formData.append(
    "new_password",
    document.getElementById("new_password").value,
  );
  formData.append(
    "new_password_repeat",
    document.getElementById("new_password_repeat").value,
  );

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
  strengh_label = document.getElementById("strengh-label");
  password_progress = document.getElementById("password-progress");

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
