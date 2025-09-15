// frontend/app.js
// Frontend application for Code Battle with Docker-based code execution support
//
// Docker Integration Features:
// - Secure code execution in isolated Docker containers
// - Multi-language support (JavaScript, Python, Go)
// - Real-time execution status and progress feedback
// - Enhanced error handling for Docker execution results
// - Execution timeout handling for long-running code
// - Language-specific code templates and syntax highlighting
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
  const languageSelector = document.getElementById("language-selector");
  const createRoomForm = document.getElementById("create-room-form");
  const roomNameInput = document.getElementById("room-name-input");
  const roomDescriptionInput = document.getElementById(
    "room-description-input",
  );
  const errorLogContainer = document.getElementById("error-log-container");
  const errorLog = document.getElementById("error-log");

  // Execution status elements
  let executionStatus = null;
  let isExecuting = false;

  // --- Monaco Editor Initialization ---
  require.config({
    paths: { vs: "https://cdn.jsdelivr.net/npm/monaco-editor@0.44.0/min/vs" },
  });
  require(["vs/editor/editor.main"], () => {
    monacoEditor = monaco.editor.create(editorContainer, {
      value: getDefaultCode("javascript"),
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
      // Clear execution status when user modifies code
      if (executionStatus) {
        executionStatus.remove();
        executionStatus = null;
      }
    });

    // Language selector change handler
    languageSelector.addEventListener("change", () => {
      const selectedLanguage = languageSelector.value;
      const defaultCode = getDefaultCode(selectedLanguage);

      // Update Monaco editor language and content
      const model = monacoEditor.getModel();
      monaco.editor.setModelLanguage(
        model,
        selectedLanguage === "go" ? "go" : selectedLanguage,
      );
      monacoEditor.setValue(defaultCode);

      // Clear any existing error messages
      if (errorLogContainer.style.display === "block") {
        errorLogContainer.style.display = "none";
      }

      // Clear execution status
      if (executionStatus) {
        executionStatus.remove();
        executionStatus = null;
      }
    });
  });

  // --- API and Logic Functions ---

  /**
   * Fetches available rooms from the backend and populates the dropdown.
   */
  // make sure to match cases, because Golang is like that
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
      (event) => {
        console.log("Correct solution event received:", event.data);

        // Show success message with execution status
        if (executionStatus) {
          executionStatus.innerHTML = `
            <div style="color: #4caf50; font-weight: bold; margin-top: 10px; padding: 10px; background-color: #e8f5e8; border-radius: 4px;">
              ✅ Code executed successfully! Solution accepted.
            </div>
          `;
          setTimeout(() => {
            if (executionStatus) {
              executionStatus.remove();
              executionStatus = null;
            }
          }, 3000);
        }

        // Clear timeout if execution completed successfully
        if (executionStatus && executionStatus.timeoutId) {
          clearTimeout(executionStatus.timeoutId);
        }

        // Reset execution state
        isExecuting = false;
        submitButton.disabled = false;
        submitButton.textContent = "Submit Solution";

        handleLeaderboardUpdate(event);
      },
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

          // Format Docker execution errors better
          logMessage = formatDockerError(logMessage);

          errorLog.textContent = logMessage;
          errorLogContainer.style.display = "block";

          // Clear timeout and remove execution status
          if (executionStatus) {
            if (executionStatus.timeoutId) {
              clearTimeout(executionStatus.timeoutId);
            }
            executionStatus.remove();
            executionStatus = null;
          }
          isExecuting = false;
          submitButton.disabled = false;
          submitButton.textContent = "Submit Solution";
        } catch (e) {
          console.error("Failed to parse wrong solution event data:", e);
          errorLog.textContent = "Failed to display error log.";
          errorLogContainer.style.display = "block";
          // Clear timeout and reset execution state on error
          if (executionStatus && executionStatus.timeoutId) {
            clearTimeout(executionStatus.timeoutId);
          }
          isExecuting = false;
          submitButton.disabled = false;
          submitButton.textContent = "Submit Solution";
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
   * Submits the user's code to the backend for Docker execution.
   *
   * This function handles:
   * - Code validation and language normalization
   * - Execution state management (preventing multiple submissions)
   * - Real-time status updates during Docker container execution
   * - Timeout handling for long-running executions
   * - Error recovery and state reset on failures
   */
  async function submitSolution() {
    if (!currentRoomId) {
      alert("Please select a room first.");
      return;
    }

    // Prevent multiple submissions
    if (isExecuting) {
      return;
    }

    // Hide any previous error messages before a new submission.
    errorLogContainer.style.display = "none";

    const code = monacoEditor.getValue().trim();

    // Basic validation
    if (!code) {
      alert("Please write some code before submitting.");
      return;
    }

    // Set execution state
    isExecuting = true;
    submitButton.disabled = true;
    submitButton.textContent = "Executing...";

    // Show execution status
    executionStatus = document.createElement("div");
    executionStatus.innerHTML = `
      <div style="color: #2196f3; margin-top: 10px; padding: 10px; background-color: #e3f2fd; border-radius: 4px;">
        🐳 Executing code in Docker container... This may take a few seconds.
      </div>
    `;
    document.getElementById("editor-section").appendChild(executionStatus);

    // Get selected language and normalize for Docker execution
    const selectedLanguage = languageSelector.value || "javascript";
    const normalizedLanguage = normalizeLanguage(selectedLanguage);

    const submission = {
      question_id: 1, // default question ID for now
      room_id: parseInt(currentRoomId, 10),
      language: normalizedLanguage,
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

      // Update execution status
      if (executionStatus) {
        executionStatus.innerHTML = `
          <div style="color: #ff9800; margin-top: 10px; padding: 10px; background-color: #fff3e0; border-radius: 4px;">
            ⏳ Code submitted to execution queue. Waiting for results...
          </div>
        `;
      }

      // Set a timeout to handle cases where Docker execution takes too long
      const executionTimeout = setTimeout(() => {
        if (isExecuting) {
          console.warn("Code execution timeout reached");

          if (executionStatus) {
            executionStatus.innerHTML = `
              <div style="color: #f44336; margin-top: 10px; padding: 10px; background-color: #ffebee; border-radius: 4px;">
                ⚠️ Execution timeout. Docker container may be busy or code is taking too long to execute.
              </div>
            `;
          }

          // Reset execution state after timeout
          setTimeout(() => {
            isExecuting = false;
            submitButton.disabled = false;
            submitButton.textContent = "Submit Solution";

            if (executionStatus) {
              executionStatus.remove();
              executionStatus = null;
            }
          }, 5000);
        }
      }, 30000); // 30 second timeout

      // Store timeout ID so we can clear it if execution completes normally
      if (executionStatus) {
        executionStatus.timeoutId = executionTimeout;
      }

      // Refresh leaderboard after submission
      setTimeout(() => fetchLeaderboard(currentRoomId), 100);
    } catch (error) {
      console.error("Failed to submit solution:", error);
      alert(`Error submitting solution: ${error.message}`);

      // Clear timeout and reset execution state on error
      if (executionStatus && executionStatus.timeoutId) {
        clearTimeout(executionStatus.timeoutId);
      }

      isExecuting = false;
      submitButton.disabled = false;
      submitButton.textContent = "Submit Solution";

      if (executionStatus) {
        executionStatus.remove();
        executionStatus = null;
      }
    }
  }

  /**
   * Normalizes language names to match Docker execution backend expectations.
   *
   * The backend Docker system expects specific language identifiers:
   * - 'js' for JavaScript execution in Node.js containers
   * - 'python' for Python execution in Python containers
   * - 'go' for Go execution in Go compiler containers
   *
   * This ensures compatibility with the Docker-based execution system.
   */
  function normalizeLanguage(language) {
    const languageMap = {
      javascript: "js",
      js: "js",
      python: "python",
      py: "python",
      golang: "go",
      go: "go",
    };

    return languageMap[language.toLowerCase()] || language;
  }

  /**
   * Returns default code template for a given language.
   *
   * Templates are designed to work with the Docker execution environment
   * and provide a starting point that will compile/run successfully.
   */
  function getDefaultCode(language) {
    const templates = {
      javascript: [
        "// Write your solution here",
        "function solve() {",
        "\treturn true;",
        "}",
      ].join("\n"),
      js: [
        "// Write your solution here",
        "function solve() {",
        "\treturn true;",
        "}",
      ].join("\n"),
      python: [
        "# Write your solution here",
        "def solve():",
        "\treturn True",
      ].join("\n"),
      go: [
        "// Write your solution here",
        "package main",
        "",
        "func solve() bool {",
        "\treturn true",
        "}",
      ].join("\n"),
    };

    return templates[language] || "// Write your solution here";
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

  /**
   * Formats Docker execution errors for better user understanding.
   *
   * This function processes error messages from Docker container execution
   * and formats them into user-friendly error messages. It handles:
   * - Common programming errors (syntax, reference, type errors)
   * - Docker-specific issues (timeouts, memory limits, container failures)
   * - Language-specific error patterns (Python indentation, Go compilation)
   * - Truncation of overly long error messages
   */
  function formatDockerError(errorMessage) {
    if (!errorMessage || errorMessage.trim() === "") {
      return "Code execution failed with no output. Please check your syntax and try again.";
    }

    // Common Docker/execution error patterns
    if (errorMessage.includes("SyntaxError")) {
      return `Syntax Error: ${errorMessage.replace(/^.*SyntaxError:?\s*/, "")}`;
    }

    if (errorMessage.includes("ReferenceError")) {
      return `Reference Error: ${errorMessage.replace(/^.*ReferenceError:?\s*/, "")}`;
    }

    if (errorMessage.includes("TypeError")) {
      return `Type Error: ${errorMessage.replace(/^.*TypeError:?\s*/, "")}`;
    }

    if (errorMessage.includes("timeout") || errorMessage.includes("SIGKILL")) {
      return "Execution timeout: Your code took too long to execute. Check for infinite loops.";
    }

    if (errorMessage.includes("memory") || errorMessage.includes("OOM")) {
      return "Memory Error: Your code used too much memory. Try optimizing your solution.";
    }

    if (errorMessage.includes("container") || errorMessage.includes("docker")) {
      return `Docker execution error: ${errorMessage}`;
    }

    if (
      errorMessage.includes("compilation failed") ||
      errorMessage.includes("compile error")
    ) {
      return `Compilation failed: ${errorMessage}`;
    }

    // Python specific errors
    if (errorMessage.includes("IndentationError")) {
      return "Python Indentation Error: Check your code indentation (use spaces or tabs consistently).";
    }

    if (errorMessage.includes("NameError")) {
      return `Python Name Error: ${errorMessage.replace(/^.*NameError:?\s*/, "")}`;
    }

    // Go specific errors
    if (
      errorMessage.includes("undefined:") ||
      errorMessage.includes("not defined")
    ) {
      return `Go Error: ${errorMessage}`;
    }

    // Return cleaned up error message
    return errorMessage.length > 200
      ? errorMessage.substring(0, 200) + "..."
      : errorMessage;
  }

  /**
   * Checks if backend Docker execution system is responsive.
   *
   * This performs a lightweight health check to determine if the
   * Docker-based code execution backend is ready to accept submissions.
   */
  async function checkExecutionHealth() {
    try {
      const response = await fetch(`${apiBaseUrl}/rooms`);
      return response.ok;
    } catch (error) {
      console.warn("Could not check backend health:", error);
      return false;
    }
  }

  /**
   * Shows Docker execution system status to user.
   *
   * Displays a warning banner if the Docker execution system appears
   * to be starting up or experiencing issues, helping users understand
   * why code execution might be slower than expected.
   */
  async function showExecutionStatus() {
    const isHealthy = await checkExecutionHealth();

    if (!isHealthy) {
      const statusElement = document.createElement("div");
      statusElement.innerHTML = `
        <div style="background-color: #fff3cd; color: #856404; padding: 10px; margin: 10px 0; border-radius: 4px; border: 1px solid #ffeaa7;">
          ⚠️ Backend execution system is starting up. Code submission may be slower initially.
        </div>
      `;
      statusElement.id = "execution-status";
      document
        .querySelector(".editor-section")
        .insertBefore(
          statusElement,
          document.querySelector(".editor-controls"),
        );

      // Check again in 10 seconds and remove warning if healthy
      setTimeout(async () => {
        const nowHealthy = await checkExecutionHealth();
        if (nowHealthy) {
          const statusEl = document.getElementById("execution-status");
          if (statusEl) {
            statusEl.remove();
          }
        }
      }, 10000);
    }
  }

  // --- Initial Load ---
  fetchRooms();
  showExecutionStatus();
});
