var error_close_buttons = document.getElementsByClassName("close_error_button");

for (const close_button of error_close_buttons) {
  close_button.addEventListener("click", function () {
    this.parentElement.classList.add("hidden");
  });
}
