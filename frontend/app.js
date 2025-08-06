document.addEventListener("DOMContentLoaded", () => {
  const apiBaseUrl = "http://localhost:8080";
  let monacoEditor;
  let currentRoomId = null;
  let leaderboardEventSource = null;

  const roomsDropdown = document.getElementById("rooms-dropdown");
  const leaderboardList = document.getElementById("leaderboard");
  const editorContainer = document.getElementById("monaco-editor");
  const submitButton = document.getElementById("submit-button");

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
   * Establishes an SSE connection to get leaderboard updates for a specific room.
   * @param {string} roomId - The ID of the room to connect to.
   */
  function connectToLeaderboard(roomId) {
    // Close any existing connection
    if (leaderboardEventSource) {
      leaderboardEventSource.close();
    }

    if (!roomId) {
      updateLeaderboard([]); // Clear leaderboard if no room is selected
      return;
    }

    console.log(`Connecting to leaderboard for room ${roomId}...`);
    leaderboardEventSource = new EventSource(
      `${apiBaseUrl}/sse/leaderboards?room_id=${roomId}`,
    );

    leaderboardEventSource.onmessage = (event) => {
      console.log("Received leaderboard update:", event.data);
      try {
        const entries = JSON.parse(event.data);
        updateLeaderboard(entries);
      } catch (error) {
        console.error("Failed to parse leaderboard data:", error);
      }
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
                <span class="player-name">${entry.player_name}</span>
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

    // Using the "true" pseudo-signal as requested
    const submission = {
      language: "javascript",
      code: "true",
    };

    console.log(`Submitting to room ${currentRoomId}:`, submission);

    try {
      // NOTE: The backend endpoint is /submission/{roomId}, but the roomId is taken
      // from a query parameter in the handler. The route should ideally be fixed
      // in the backend, but we'll use a query param to match the current backend implementation.
      const response = await fetch(
        `${apiBaseUrl}/submission/${currentRoomId}?roomId=${currentRoomId}`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(submission),
        },
      );

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`Submission failed: ${errorText}`);
      }

      // The current backend handler doesn't return a JSON response on success,
      // so we just check for the OK status.
      console.log("Solution submitted successfully!");
      alert("Solution submitted!");
    } catch (error) {
      console.error("Failed to submit solution:", error);
      alert(`Error submitting solution: ${error.message}`);
    }
  }

  // --- Event Listeners ---

  roomsDropdown.addEventListener("change", () => {
    currentRoomId = roomsDropdown.value;
    submitButton.disabled = !currentRoomId; // Enable button only if a room is selected
    connectToLeaderboard(currentRoomId);
  });

  submitButton.addEventListener("click", submitSolution);

  // --- Initial Load ---
  fetchRooms();
});
