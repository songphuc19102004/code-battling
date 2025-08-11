-- -- Players
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


-- Programming Languages
-- name: CreateProgrammingLanguage :one
INSERT INTO programming_languages (id, name)
VALUES ($1, $2)
RETURNING *;

-- name: GetProgrammingLanguage :one
SELECT * FROM programming_languages
WHERE id = $1;

-- name: ListProgrammingLanguages :many
SELECT * FROM programming_languages
ORDER BY id;

-- name: UpdateProgrammingLanguage :one
UPDATE programming_languages
SET name = $2
WHERE id = $1
RETURNING *;

-- name: DeleteProgrammingLanguage :exec
DELETE FROM programming_languages
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
