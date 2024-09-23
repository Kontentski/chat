document.addEventListener("DOMContentLoaded", function() {
  // Handle form submission
  document
    .getElementById("register-form")
    .addEventListener("submit", async function (event) {
      event.preventDefault();

      const username = document.getElementById("username").value;
      const password = document.getElementById("password").value;
      const errorMessageDiv = document.getElementById("error-message");

      // Clear any previous error messages
      errorMessageDiv.textContent = "";

      // Prepare form data
      const formData = new URLSearchParams();
      formData.append("username", username);
      formData.append("password", password);

      try {
        // Send POST request to the server
        const response = await fetch("/auth/register", {
          method: "POST",
          body: formData,
          headers: {
            "Content-Type": "application/x-www-form-urlencoded",
          },
        });

        const data = await response.json();

        if (response.ok) {
          // Redirect to home page on successful registration
          window.location.href = "/homepage";
        } else {
          // Display error message from server
          errorMessageDiv.textContent = data.error;
        }
      } catch (error) {
        console.error("Error:", error);
        errorMessageDiv.textContent = "An error occurred. Please try again.";
      }
    });

  // Toggle password visibility
  document.getElementById('show-password').addEventListener('click', function() {
    const passwordInput = document.getElementById('password');
    if (passwordInput.type === 'password') {
      passwordInput.type = 'text';  // Show the password
    } else {
      passwordInput.type = 'password';  // Hide the password
    }
  });
});
