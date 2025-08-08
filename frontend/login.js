document.addEventListener("DOMContentLoaded", () => {
  const apiBaseUrl = "http://localhost:8080";
  const authForm = document.getElementById("auth-form");
  const nameInput = document.getElementById("player-name-input");
  const passwordInput = document.getElementById("player-password-input");
  const submitButton = document.getElementById("submit-button");
  const toggleLink = document.getElementById("toggle-form-link");
  const formTitle = document.getElementById("form-title");
  const formSubtitle = document.getElementById("form-subtitle");

  let isLoginMode = true; // Start in login mode by default

  // If the user is already logged in, redirect them to the main app
  if (localStorage.getItem("codeBattlePlayer")) {
    window.location.href = "index.html";
    return; // Stop the rest of the script from running
  }

  function updateFormUI() {
    if (isLoginMode) {
      formTitle.textContent = "Login to Code Battle";
      formSubtitle.textContent = "Enter your credentials to join the arena.";
      submitButton.textContent = "Login";
      toggleLink.textContent = "Need an account? Register here.";
    } else {
      formTitle.textContent = "Register for Code Battle";
      formSubtitle.textContent = "Create an account to start competing.";
      submitButton.textContent = "Register";
      toggleLink.textContent = "Already have an account? Login here.";
    }
  }

  toggleLink.addEventListener("click", (e) => {
    e.preventDefault();
    isLoginMode = !isLoginMode;
    updateFormUI();
  });

  authForm.addEventListener("submit", async (event) => {
    event.preventDefault();

    const playerName = nameInput.value.trim();
    const password = passwordInput.value.trim();

    if (!playerName || !password) {
      alert("Please enter both your name and password.");
      return;
    }

    const endpoint = isLoginMode ? "/players/login" : "/players";
    const payload = {
      name: playerName,
      password: password,
    };

    try {
      const response = await fetch(`${apiBaseUrl}${endpoint}`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      });

      const result = await response.json();

      if (!response.ok) {
        throw new Error(result.message || "An error occurred.");
      }

      const playerData = result.data;

      localStorage.setItem("codeBattlePlayer", JSON.stringify(playerData));
      window.location.href = "index.html";
    } catch (error) {
      console.error("Authentication failed:", error);
      alert(`Could not complete action: ${error.message}`);
    }
  });

  // Initialize the UI
  updateFormUI();
});
