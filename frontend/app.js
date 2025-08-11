// frontend/app.js
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
  const leaveRoomButton = document.getElementById("leave-room-button");
  const createRoomForm = document.getElementById("create-room-form");
  const roomNameInput = document.getElementById("room-name-input");
  const roomDescriptionInput = document.getElementById(
    "room-description-input",
  );
  const errorLogContainer = document.getElementById("error-log-container");
  const errorLog = document.getElementById("error-log");

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

    // Clear error log when user starts typing
    monacoEditor.onDidChangeModelContent(() => {
      if (errorLogContainer.style.display === "block") {
        errorLogContainer.style.display = "none";
      }
    });
  });

  // --- API and Logic Functions ---

  /**
   * Fetches available rooms from the backend and populates the dropdown.
   */
  async function fetchRooms() {
    console.log("hit fetchrooms()");
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
        console.log(room);
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

    const handleLeaderboardUpdate = (event) => {
      console.log(
        `Leaderboard-updating event received: ${event.type}. Refreshing leaderboard.`,
      );
      console.log("Event data:", event.data);
      fetchLeaderboard(roomId);
    };

    // These events indicate that the leaderboard state has changed.
    leaderboardEventSource.addEventListener(
      "CORRECT_SOLUTION_SUBMITTED",
      handleLeaderboardUpdate,
    );
    leaderboardEventSource.addEventListener(
      "PLAYER_JOINED",
      handleLeaderboardUpdate,
    );
    leaderboardEventSource.addEventListener(
      "PLAYER_LEFT",
      handleLeaderboardUpdate,
    );

    // Handle wrong submissions specifically to show logs
    leaderboardEventSource.addEventListener(
      "WRONG_SOLUTION_SUBMITTED",
      (event) => {
        console.log("Wrong solution event received:", event.data);
        try {
          const eventPayload = JSON.parse(event.data);

          // The backend sends the log in the format "log:THE_ACTUAL_LOG"
          // in the `Data` field of the event payload.
          let logMessage = "An unknown error occurred.";
          if (eventPayload && typeof eventPayload.Data === "string") {
            logMessage = eventPayload.Data.startsWith("log:")
              ? eventPayload.Data.substring(4)
              : eventPayload.Data;
          }

          errorLog.textContent = logMessage;
          errorLogContainer.style.display = "block";
        } catch (e) {
          console.error("Failed to parse wrong solution event data:", e);
          errorLog.textContent = "Failed to display error log.";
          errorLogContainer.style.display = "block";
        }
      },
    );

    // This event indicates a room was removed, so we need to update the room list.
    leaderboardEventSource.addEventListener("ROOM_DELETED", (event) => {
      console.log("Room deleted event received:", event.data);
      alert("A room has been deleted. The interface will now refresh.");

      // Reset the current room selection
      currentRoomId = null;
      roomsDropdown.value = "";
      submitButton.disabled = true;
      leaveRoomButton.style.display = "none";
      updateLeaderboard([]); // Clear leaderboard

      // Refresh the list of available rooms
      fetchRooms();
    });

    leaderboardEventSource.onerror = (error) => {
      console.error(
        "SSE connection error. The browser will attempt to reconnect automatically.",
        error,
      );
      // The browser will handle reconnection automatically.
    };
  }

  /**
   * Renders the leaderboard in the UI.
   * @param {Array} entries - An array of leaderboard entry objects.
   */

  function updateLeaderboard(entries) {
    // Always clear the current list first to prevent duplicates.
    leaderboardList.innerHTML = "";

    if (!entries || entries.length === 0) {
      leaderboardList.innerHTML = "<li>No leaderboard data available.</li>";
      return;
    }

    // Create a document fragment to build the new list in memory.
    const fragment = document.createDocumentFragment();

    entries.forEach((entry) => {
      const li = document.createElement("li");
      li.innerHTML = `
                <span class="player-place">${entry.place}.</span>
                <span class="player-name">${entry.player_name}</span>
                <span class="player-score">${entry.score} points</span>
            `;
      fragment.appendChild(li);
    });

    // Append the entire new list at once.
    leaderboardList.appendChild(fragment);
  }

  /**
   * Submits the user's code to the backend.
   */
  async function submitSolution() {
    if (!currentRoomId) {
      alert("Please select a room first.");
      return;
    }

    // Hide any previous error messages before a new submission.
    errorLogContainer.style.display = "none";

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

      // Refresh leaderboard after submission
      setTimeout(() => fetchLeaderboard(currentRoomId), 100);
    } catch (error) {
      console.error("Failed to submit solution:", error);
      alert(`Error submitting solution: ${error.message}`);
    }
  }

  /**
   * Handles leaving the current room.
   */
  async function leaveRoom() {
    if (!currentRoomId) {
      alert("No room selected to leave.");
      return;
    }

    const confirmation = confirm(
      "Are you sure you want to leave this room? Your progress in this room will be lost.",
    );

    if (!confirmation) {
      return;
    }

    try {
      const response = await fetch(
        `${apiBaseUrl}/rooms/${currentRoomId}/players/${currentPlayer.id}`,
        {
          method: "DELETE",
        },
      );

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || "Failed to leave room.");
      }

      console.log("Successfully left the room");

      // Reset the UI state
      currentRoomId = null;
      roomsDropdown.value = "";
      submitButton.disabled = true;
      leaveRoomButton.style.display = "none";
      updateLeaderboard([]);

      // Close the SSE connection
      if (leaderboardEventSource) {
        leaderboardEventSource.close();
        leaderboardEventSource = null;
      }

      alert("You have successfully left the room.");
    } catch (error) {
      console.error("Failed to leave room:", error);
      alert(`Error leaving room: ${error.message}`);
    }
  }

  // --- Event Listeners ---

  roomsDropdown.addEventListener("change", () => {
    currentRoomId = roomsDropdown.value;
    submitButton.disabled = !currentRoomId; // Enable button only if a room is selected

    if (currentRoomId) {
      // Show the leave room button when a room is selected
      leaveRoomButton.style.display = "block";

      // Fetch initial leaderboard data
      fetchLeaderboard(currentRoomId);
      // Connect to SSE for real-time updates
      connectToRoomEvents(currentRoomId);
    } else {
      // Hide the leave room button when no room is selected
      leaveRoomButton.style.display = "none";
      updateLeaderboard([]);
    }
  });

  submitButton.addEventListener("click", submitSolution);

  leaveRoomButton.addEventListener("click", leaveRoom);

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
