-- +goose Up
-- +goose StatementBegin
-- migrations.sql

CREATE TABLE IF NOT EXISTS chat_rooms (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    description TEXT,
    type VARCHAR(50) 
);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100),
    password VARCHAR(100) NOT NULL,
    email VARCHAR(100) NOT NULL UNIQUE,
    profile_picture TEXT,
    last_seen TIMESTAMP,
    created_at TIMESTAMP DEFAULT current_timestamp
);

CREATE TABLE IF NOT EXISTS messages (
    message_id SERIAL,
    chat_room_id INT,
    sender_id INT,
    content TEXT NOT NULL,
    timestamp TIMESTAMP DEFAULT current_timestamp,
    is_dm BOOLEAN,
    PRIMARY KEY (message_id, chat_room_id),
    FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (chat_room_id) REFERENCES chat_rooms(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS chat_room_members (
    chat_room_id INT,
    user_id INT,
    PRIMARY KEY (chat_room_id, user_id),
    FOREIGN KEY (chat_room_id) REFERENCES chat_rooms(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS read_messages (
    user_id INT,
    message_id INT,
    chat_room_id INT,
    read_at TIMESTAMP NULL,
    PRIMARY KEY (user_id, message_id, chat_room_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (message_id, chat_room_id) REFERENCES messages(message_id, chat_room_id) ON DELETE CASCADE,
    FOREIGN KEY (chat_room_id) REFERENCES chat_rooms(id) ON DELETE CASCADE
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users;

DROP TABLE messages;

DROP TABLE chat_rooms;

DROP TABLE chat_room_members;

DROP TABLE read_messages;

-- +goose StatementEnd
