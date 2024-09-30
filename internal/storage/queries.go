package storage

const (
	CreateUserQuery = `
		INSERT INTO users (username, name, password, email, profile_picture, created_at) 
		VALUES ($1, $2, $3, $4, $5, NOW()) 
		RETURNING id`

	IsUserInChatRoomQuery = `
		SELECT COUNT(*) 
		FROM chat_room_members 
		WHERE user_id = $1 AND chat_room_id = $2
		`
// TODO: make isuserexist to use %like operator
	IsUserExistsQuery = `
	SELECT COUNT(*)
	FROM users
	WHERE username = $1
	`

	DeleteMessageQuery = `DELETE FROM messages WHERE message_id=$1 AND chat_room_id=$2`

	GetMessagesQuery = "SELECT m.message_id, m.sender_id, u.username, u.name, m.content, m.timestamp, m.chat_room_id, m.is_dm, COALESCE(r.read_at, '1970-01-01T00:00:00Z') AS read_at FROM messages m JOIN users u ON m.sender_id = u.id LEFT JOIN read_messages r ON m.message_id = r.message_id AND r.user_id = $1 AND m.chat_room_id = r.chat_room_id WHERE m.chat_room_id = $2 ORDER BY m.timestamp ASC"


	FetchUserChatRoomsQuery = `
	SELECT cr.id, cr.name, cr.description, cr.type
	FROM chat_rooms cr
	JOIN chat_room_members crm ON cr.id = crm.chat_room_id
	WHERE crm.user_id = $1
	`

	DeleteUserFromChatRoomQuery = `
	DELETE FROM chat_room_members 
	WHERE user_id = $1 AND chat_room_id = $2
	`
)
