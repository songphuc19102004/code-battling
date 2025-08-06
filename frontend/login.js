document.addEventListener("DOMContentLoaded", () => {
    const apiBaseUrl = "http://localhost:8080";
    const loginForm = document.getElementById("login-form");
    const nameInput = document.getElementById("player-name-input");

    // If the user is already logged in, redirect them to the main app
    if (localStorage.getItem("codeBattlePlayer")) {
        window.location.href = "index.html";
        return; // Stop the rest of the script from running
    }

    loginForm.addEventListener("submit", async (event) => {
        event.preventDefault(); // Prevent the form from causing a page reload

        const playerName = nameInput.value.trim();
        if (!playerName) {
            alert("Please enter your name.");
            return;
        }

        try {
            // This endpoint will be created in the Go backend next
            const response = await fetch(`${apiBaseUrl}/players`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({ name: playerName }),
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.message || "Failed to create player.");
            }

            const result = await response.json();
            const playerData = result.data; // Expecting { id: <number>, name: <string> }

            // Save the player data to localStorage to persist the "session"
            localStorage.setItem("codeBattlePlayer", JSON.stringify(playerData));

            // Redirect to the main application page on successful login
            window.location.href = "index.html";

        } catch (error) {
            console.error("Login failed:", error);
            alert(`Could not log in: ${error.message}`);
        }
    });
});
