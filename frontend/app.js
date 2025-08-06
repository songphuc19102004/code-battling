document.addEventListener("DOMContentLoaded", () => {
  const apiBaseUrl = "http://localhost:8080";
  let monacoEditor;
  let currentRoomId = null;
  let leaderboardEventSource = null;
  let currentPlayer = null;

  // --- Authentication Check ---
  const playerData = localStorage.getItem("codeBattlePlayer");
  if (!playerData) {
    // If no player data is found, redirect to the login page.
    window.location.href = "login.html";
    return; // Stop the script from executing further.
  } else {
    currentPlayer = JSON.parse(playerData);
    console.log(`Welcome, ${currentPlayer.name} (ID: ${currentPlayer.id})`);
  }

  const roomsDropdown = document.getElementById("rooms-dropdown");
  const leaderboardList = document.getElementById("leaderboard");
  const editorContainer = document.getElementById("monaco-editor");
  const submitButton = document.getElementById("submit-button");
  const createRoomForm = document.getElementById("create-room-form");
  const roomNameInput = document.getElementById("room-name-input");
  const roomDescriptionInput = document.getElementById(
    "room-description-input",
  );

  // --- Monaco Editor Initialization ---
  require.config({
    paths: { vs: "https://cdn.jsdelivr.net/npm/monaco-editor@0.44.0/min/vs" },
  });
  require(["vs/editor/editor.main"], () => {
    monacoEditor = monaco.editor.create(editorContainer, {
      value: [
        "// Write your solution here",
        "function solve() {",
        "\treturn true;",
        "}",
      ].join("\n"),
      language: "javascript",
      theme: "vs-dark",
      automaticLayout: true,
    });
    // Initially disable the submit button until a room is selected
    submitButton.disabled = true;
  });

  // --- API and Logic Functions ---

  /**
   * Fetches available rooms from the backend and populates the dropdown.
   */
  async function fetchRooms() {
    try {
      const response = await fetch(`${apiBaseUrl}/rooms`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const roomsResponse = await response.json();
      const rooms = roomsResponse.data;

      // Clear existing options except the placeholder
      roomsDropdown.innerHTML = '<option value="">--Select a Room--</option>';

      for (const roomId in rooms) {
        const room = rooms[roomId];
        const option = document.createElement("option");
        option.value = room.id;
        option.textContent = room.name;
        roomsDropdown.appendChild(option);
      }
    } catch (error) {
      console.error("Failed to fetch rooms:", error);
      alert("Could not fetch rooms. Is the backend server running?");
    }
  }

  /**
   * Fetches leaderboard data for a specific room.
   * @param {string} roomId - The ID of the room.
   */
  async function fetchLeaderboard(roomId) {
    if (!roomId) {
      updateLeaderboard([]); // Clear leaderboard if no room is selected
      return;
    }

    try {
      const response = await fetch(`${apiBaseUrl}/rooms/${roomId}/leaderboard`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const leaderboardResponse = await response.json();
      const entries = leaderboardResponse.data?.entries || [];
      updateLeaderboard(entries);
    } catch (error) {
      console.error("Failed to fetch leaderboard:", error);
      updateLeaderboard([]); // Clear leaderboard on error
    }
  }

  /**
   * Establishes an SSE connection to get real-time events for a specific room.
   * @param {string} roomId - The ID of the room to connect to.
   */
  function connectToRoomEvents(roomId) {
    // Close any existing connection
    if (leaderboardEventSource) {
      leaderboardEventSource.close();
    }

    if (!roomId) {
      return;
    }

    console.log(`Connecting to room events for room ${roomId}...`);
    leaderboardEventSource = new EventSource(
      `${apiBaseUrl}/events?room_id=${roomId}&player_id=${currentPlayer.id}`,
    );

    leaderboardEventSource.onmessage = (event) => {
      console.log("Received room event:", event.data);
      // When we receive an event, refresh the leaderboard
      fetchLeaderboard(roomId);
    };

    leaderboardEventSource.onerror = (error) => {
      console.error(
        "SSE connection error. The browser will attempt to reconnect automatically.",
        error,
      );
      // By not calling close(), we allow the browser to handle reconnection attempts.
    };
  }

  /**
   * Renders the leaderboard in the UI.
   * @param {Array} entries - An array of leaderboard entry objects.
   */
  function updateLeaderboard(entries) {
    leaderboardList.innerHTML = ""; // Clear current list
    if (!entries || entries.length === 0) {
      leaderboardList.innerHTML = "<li>No leaderboard data available.</li>";
      return;
    }

    entries.forEach((entry) => {
      const li = document.createElement("li");
      li.innerHTML = `
                <span class="player-place">${entry.place}.</span>
                <span class="player-name">Player ${entry.player_id}</span>
                <span class="player-score">${entry.score} points</span>
            `;
      leaderboardList.appendChild(li);
    });
  }

  /**
   * Submits the user's code to the backend.
   */
  async function submitSolution() {
    if (!currentRoomId) {
      alert("Please select a room first.");
      return;
    }

    const code = monacoEditor.getValue();

    const submission = {
      question_id: 1, // default question ID for now
      room_id: parseInt(currentRoomId, 10),
      language: "javascript",
      code: code,
      player_id: currentPlayer.id,
      submitted_at: new Date().toISOString(),
    };

    console.log(
      `Submitting for player ${currentPlayer.name} to room ${currentRoomId}:`,
      submission,
    );

    try {
      const response = await fetch(`${apiBaseUrl}/submission`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(submission),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`Submission failed: ${errorText}`);
      }

      console.log("Solution submitted successfully!");
      alert("Solution submitted!");

      // Refresh leaderboard after submission
      setTimeout(() => fetchLeaderboard(currentRoomId), 1000);
    } catch (error) {
      console.error("Failed to submit solution:", error);
      alert(`Error submitting solution: ${error.message}`);
    }
  }

  // --- Event Listeners ---

  roomsDropdown.addEventListener("change", () => {
    currentRoomId = roomsDropdown.value;
    submitButton.disabled = !currentRoomId; // Enable button only if a room is selected

    if (currentRoomId) {
      // Fetch initial leaderboard data
      fetchLeaderboard(currentRoomId);
      // Connect to SSE for real-time updates
      connectToRoomEvents(currentRoomId);
    } else {
      updateLeaderboard([]);
    }
  });

  submitButton.addEventListener("click", submitSolution);

  /**
   * Handles the submission of the create room form.
   * @param {Event} event - The form submission event.
   */
  async function handleCreateRoom(event) {
    event.preventDefault(); // Prevent the default form submission which reloads the page

    const name = roomNameInput.value.trim();
    const description = roomDescriptionInput.value.trim();

    if (!name || !description) {
      alert("Room name and description cannot be empty.");
      return;
    }

    try {
      const response = await fetch(`${apiBaseUrl}/rooms`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ name, description }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || "Failed to create room.");
      }

      const result = await response.json();
      alert(`Room "${result.data.name}" created successfully!`);

      // Clear the form and refresh the rooms list
      createRoomForm.reset();
      await fetchRooms();
    } catch (error) {
      console.error("Failed to create room:", error);
      alert(`Error creating room: ${error.message}`);
    }
  }

  createRoomForm.addEventListener("submit", handleCreateRoom);

  // --- Initial Load ---
  fetchRooms();
});
