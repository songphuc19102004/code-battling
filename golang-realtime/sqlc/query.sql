-- Players
-- name: CreatePlayer :one
INSERT INTO players (id, name, password)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetPlayer :one
SELECT * FROM players
WHERE id = $1;

-- name: GetPlayerByName :one
SELECT * FROM players
WHERE name = $1;

-- name: ListPlayers :many
SELECT * FROM players
ORDER BY id;

-- name: UpdatePlayer :one
UPDATE players
SET name = $2, password = $3
WHERE id = $1
RETURNING *;

-- name: DeletePlayer :exec
DELETE FROM players
WHERE id = $1;

-- Rooms
-- name: CreateRoom :one
INSERT INTO rooms (id, name, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRoom :one
SELECT * FROM rooms
WHERE id = $1;

-- name: ListRooms :many
SELECT * FROM rooms
ORDER BY id;

-- name: UpdateRoom :one
UPDATE rooms
SET name = $2, description = $3
WHERE id = $1
RETURNING *;

-- name: DeleteRoom :exec
DELETE FROM rooms
WHERE id = $1;


-- Languages
-- name: CreateLanguage :one
INSERT INTO languages (id, name, compile_cmd, run_cmd, timeout_second)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetLanguage :one
SELECT * FROM languages
WHERE id = $1;

-- name: GetLanguageByName :one
SELECT * FROM languages
where name = $1;

-- name: ListLanguages :many
SELECT * FROM languages
ORDER BY id;

-- name: UpdateLanguage :one
UPDATE languages
SET name = $2, compile_cmd = $3, run_cmd = $4, timeout_second = $5
WHERE id = $1
RETURNING *;

-- name: DeleteLanguage :exec
DELETE FROM languages
WHERE id = $1;


-- Questions
-- name: CreateQuestion :one
INSERT INTO questions (id, language_id, template_function, title, description, score, difficulty)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetQuestion :one
SELECT * FROM questions
WHERE id = $1 AND language_id = $2;

-- name: ListQuestions :many
SELECT * FROM questions
ORDER BY id, language_id;

-- name: ListQuestionsByLanguage :many
SELECT * FROM questions
WHERE language_id = $1
ORDER BY id;

-- name: UpdateQuestion :one
UPDATE questions
SET template_function = $3, title = $4, description = $5, score = $6, difficulty = $7
WHERE id = $1 AND language_id = $2
RETURNING *;

-- name: DeleteQuestion :exec
DELETE FROM questions
WHERE id = $1 AND language_id = $2;


-- Test Cases
-- name: CreateTestCase :one
INSERT INTO test_cases (question_id, question_language_id, input, expected_output, time_constraint, space_constraint)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetTestCase :one
SELECT * FROM test_cases
WHERE id = $1;

-- name: ListTestCasesForQuestion :many
SELECT * FROM test_cases
WHERE question_id = $1 AND question_language_id = $2
ORDER BY id;

-- name: UpdateTestCase :one
UPDATE test_cases
SET input = $2, expected_output = $3, time_constraint = $4, space_constraint = $5
WHERE id = $1
RETURNING *;

-- name: DeleteTestCase :exec
DELETE FROM test_cases
WHERE id = $1;


-- Room Players
-- name: CreateRoomPlayer :one
INSERT INTO room_players (room_id, player_id, score, place)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetRoomPlayer :one
SELECT * FROM room_players
WHERE room_id = $1 AND player_id = $2;

-- name: ListPlayersInRoom :many
SELECT p.* FROM players p
JOIN room_players rp ON p.id = rp.player_id
WHERE rp.room_id = $1
ORDER BY rp.place;

-- name: UpdateRoomPlayerScore :one
UPDATE room_players
SET score = $3, place = $4
WHERE room_id = $1 AND player_id = $2
RETURNING *;

-- name: GetRoomPlayers :many
SELECT rp.*
FROM room_players rp
WHERE rp.room_id = $1
ORDER BY rp.score DESC;

-- name: AddRoomPlayerScore :one
UPDATE room_players
SET score = score + sqlc.arg(score_too_add)
WHERE room_id = $1 AND player_id = $2
RETURNING *;

-- name: DeleteRoomPlayer :exec
DELETE FROM room_players
WHERE room_id = $1 AND player_id = $2;

-- name: GetLeaderboardForRoom :many
SELECT p.name, rp.score, rp.place
FROM room_players rp
JOIN players p ON rp.player_id = p.id
WHERE rp.room_id = $1
ORDER BY rp.place;

-- name: UpdateRoomPlayerRanks :exec
WITH ranked_players AS (
  SELECT
    player_id,
    RANK() OVER (ORDER BY score DESC) as new_place
  FROM room_players
  WHERE room_id = $1
)
UPDATE room_players rp
SET place = rp_ranked.new_place
FROM ranked_players rp_ranked
WHERE rp.room_id = $1 AND rp.player_id = rp_ranked.player_id;

-- Submissions
-- name: CreateSubmission :one
INSERT INTO submissions (source_code, language_id, stdin, expected_output, stdout, status_id, created_at, finished_at, time, memory, stderr, token, number_of_runs, cpu_time_limit, cpu_extra_time, wall_time_limit, memory_limit, stack_limit, max_processes_and_or_threads, enable_per_process_and_thread_time_limit, enable_per_process_and_thread_memory_limit, max_file_size, compile_output, exit_code, exit_signal, message, wall_time, compiler_options, command_line_arguments, redirect_stderr_to_stdout, callback_url, additional_files, enable_network, started_at, queued_at, updated_at, queue_host, execution_host)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38)
RETURNING *;

-- name: GetSubmission :one
SELECT * FROM submissions
WHERE id = $1;

-- name: ListSubmissions :many
SELECT * FROM submissions
ORDER BY id;

-- name: UpdateSubmission :one
UPDATE submissions
SET source_code = $2, language_id = $3, stdin = $4, expected_output = $5, stdout = $6, status_id = $7, created_at = $8, finished_at = $9, time = $10, memory = $11, stderr = $12, token = $13, number_of_runs = $14, cpu_time_limit = $15, cpu_extra_time = $16, wall_time_limit = $17, memory_limit = $18, stack_limit = $19, max_processes_and_or_threads = $20, enable_per_process_and_thread_time_limit = $21, enable_per_process_and_thread_memory_limit = $22, max_file_size = $23, compile_output = $24, exit_code = $25, exit_signal = $26, message = $27, wall_time = $28, compiler_options = $29, command_line_arguments = $30, redirect_stderr_to_stdout = $31, callback_url = $32, additional_files = $33, enable_network = $34, started_at = $35, queued_at = $36, updated_at = $37, queue_host = $38, execution_host = $39
WHERE id = $1
RETURNING *;

-- name: DeleteSubmission :exec
DELETE FROM submissions
WHERE id = $1;

--name:
