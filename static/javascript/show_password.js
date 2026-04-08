const showPasswordButtons = document.getElementsByClassName(
  "show_password_toggle",
);

for (const button of showPasswordButtons) {
  button.addEventListener("click", function () {
    const input = this.parentElement.querySelector(
      "input[type='password'], input[type='text']",
    );
    if (!input) return;
    const isHidden = input.type === "password";
    input.type = isHidden ? "text" : "password";

    if (isHidden) {
      this.classList.add("open");
      this.classList.remove("closed");
    } else {
      this.classList.remove("open");
      this.classList.add("closed");
    }
  });
}
