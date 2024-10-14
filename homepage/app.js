let username;
let userID;
let socket;
let currentChatRoomID = null;
let pingInterval = 60000; // 60 seconds
let titleInterval = null;

const notificationSound = new Audio("assets/notification.mp3");
const chatRoomList = document.getElementById("chat-room-list-items");
const chatBox = document.getElementById("chat-box");

document.addEventListener("DOMContentLoaded", function () {
	const loginButton = document.getElementById("google-login");
	const logoutButton = document.getElementById("logout-button");
	const userInfo = document.getElementById("user-info");

	console.log("dom loaded");
	// Utility function to get cookie value
	function getCookie(name) {
		const value = `; ${document.cookie}`;
		const parts = value.split(`; ${name}=`);
		if (parts.length === 2) return parts.pop().split(";").shift();
	}

	function checkLoginStatus() {
		const authSession = getCookie("auth-session");

		if (authSession) {
			loginButton.style.display = "none";
			logoutButton.style.display = "block";
			userInfo.style.display = "block";
		} else {
			loginButton.style.display = "block";
			logoutButton.style.display = "none";
			userInfo.style.display = "block";
		}
	}

	checkLoginStatus();

	logoutButton.addEventListener("click", function () {
		fetch("/auth/logout", { method: "POST" }) // Send POST request to your logout route
			.then((response) => {
				if (response.ok) {
					document.cookie = "auth-session=; Max-Age=0; path=/"; // Clear auth-session cookie
					window.location.reload(); // Refresh the page to show login state
				} else {
					alert("Logout failed");
				}
			});
	});
});

// Google login button
document.getElementById("google-login").addEventListener("click", function () {
	window.location.href =
		"https://champion-thoroughly-walrus.ngrok-free.app/auth?provider=google";
});

// Utility function to get cookie value
function getCookie(name) {
	const value = `; ${document.cookie}`;
	const parts = value.split(`; ${name}=`);
	if (parts.length === 2) return parts.pop().split(";").shift();
}

// Connect to WebSocket server with username
function connectWebSocket() {
	const wsUrl = `wss://champion-thoroughly-walrus.ngrok-free.app/ws`;
	socket = new WebSocket(wsUrl);

	socket.onopen = function () {
		console.log("Connected to WebSocket server");
	};

	socket.onmessage = function (event) {
		const data = JSON.parse(event.data);
		console.log("Received data:", data);

		if (data.userID) {
			handleUserID(data);
		} else if (Array.isArray(data)) {
			handleChatRooms(data);
		} else if (data.type === "delete") {
			handleDeleteMessage(data.message_id, data.chat_room_id);
		} else if (data.type === "image" || data.type === "video") {
			handleMediaMessage(data);
		} else if (data.chat_room_id === currentChatRoomID) {
			handleIncomingMessage(data);
		} else {
			handleOtherChatRoomMessages(data);
		}
	};

	socket.onclose = function (event) {
		console.log("WebSocket connection closed", event);
		clearInterval(pingInterval); // Clear ping interval
		reconnectWebSocket(); // Attempt to reconnect
	};

	socket.onerror = function (error) {
		console.error("WebSocket error:", error);
	};
}

connectWebSocket();

// message deletion
function handleDeleteMessage(messageID, chatRoomID) {
	const chatRoomListItem = document.querySelector(
		`li[data-chat-room-id="${chatRoomID}"]`
	);

	if (chatRoomListItem) {
		// Use the chat room ID to find the associated chat box container
		const chatBox = document.getElementById("chat-box");

		// Ensure that the chatBox contains messages for the chatRoomID
		if (chatBox) {
			const messageElement = chatBox.querySelector(
				`[data-message-id="${messageID}"]`
			);

			if (messageElement) {
				console.log("message deleted");
				messageElement.remove();
			}
		}
	} else {
		console.error("Chat room not found for ID:", chatRoomID);
	}
}

function reconnectWebSocket() {
	console.log("Attempting to reconnect WebSocket...");
	setTimeout(() => {
		connectWebSocket();
	}, 5000); // Retry after 5 seconds
}

// Handle user ID reception
function handleUserID(data) {
	userID = data.userID;
	document.getElementById(
		"user-username"
	).textContent = `Username: ${data.username}`;
	document.getElementById("user-name").textContent = `Name: ${data.name}`;
	fetchChatRooms();
}

// Handle chat rooms data
function handleChatRooms(chatRooms) {
	chatRoomList.innerHTML = "";

	// Track the current selected chat room
	let currentChatRoomId = null;

	chatRooms.forEach((room) => {
		const listItem = document.createElement("li");
		listItem.dataset.chatRoomId = room.id;
		listItem.style.position = "relative"; // Enable positioning for dropdown

		// Create a clickable element for the chat room name
		const chatRoomName = document.createElement("span");
		chatRoomName.textContent = room.name;
		chatRoomName.classList.add("chat-room-name");
		chatRoomName.style.cursor = "pointer"; // Change cursor to pointer for clickable effect

		// Create settings button (three dots), hidden initially
		const settingsButton = document.createElement("button");
		settingsButton.textContent = "⋮"; // Three dots icon
		settingsButton.classList.add("settings-button");
		settingsButton.style.display = "none"; // Hidden by default
		settingsButton.style.marginLeft = "10px";

		// Create dropdown container
		const dropdown = document.createElement("div");
		dropdown.classList.add("dropdown-content");
		dropdown.style.display = "none"; // Initially hidden

		// Add Members to Group option
		const addMembers = document.createElement("button");
		addMembers.textContent = "Add Members to Group";
		addMembers.addEventListener("click", () => addMembersToGroup(room.id));

		// Leave Group option
		// Leave Group option
		const leaveGroup = document.createElement("button");
		leaveGroup.textContent = "Leave Group";
		leaveGroup.dataset.chatRoomId = room.id; // Ensure ID is set
		leaveGroup.dataset.chatRoomName = room.name; // Set the name correctly
		leaveGroup.addEventListener("click", () =>
			leaveChatRoom(room.id, room.name)
		);

		// Append options to the dropdown
		dropdown.appendChild(addMembers);
		dropdown.appendChild(leaveGroup);

		// Show dropdown on settings button click
		settingsButton.addEventListener("click", (e) => {
			e.stopPropagation(); // Prevent click event from triggering other elements

			dropdown.style.display =
				dropdown.style.display === "block" ? "none" : "block";

			// Get the position of the settings button
			const rect = settingsButton.getBoundingClientRect();
		});

		// Close dropdown if clicked outside
		document.addEventListener("click", (e) => {
			if (!listItem.contains(e.target)) {
				dropdown.style.display = "none";
			}
		});

		// Event listener for clicking on the chat room name
		chatRoomName.addEventListener("click", () => {
			selectChatRoom(room.id);

			// Show the settings button for the selected chatroom
			if (currentChatRoomId !== null) {
				// Hide settings button for the previous chatroom
				const previousChatRoom = document.querySelector(
					`li[data-chat-room-id="${currentChatRoomId}"] .settings-button`
				);
				if (previousChatRoom) {
					previousChatRoom.style.display = "none";
				}
			}

			// Update the current chat room ID
			currentChatRoomId = room.id;

			// Show the settings button for the newly selected chatroom
			settingsButton.style.display = "inline-block";
		});

		// Append elements
		listItem.appendChild(chatRoomName); // Append chat room name first
		listItem.appendChild(settingsButton); // Then append settings button (hidden initially)
		listItem.appendChild(dropdown); // Append dropdown to the list item
		chatRoomList.appendChild(listItem);
	});
}

// Handle incoming message for the current chat room
function handleIncomingMessage(message) {
	appendMessageToChatBox(message);
	if (document.hidden) {
		playNotificationSound();
		startFlashingTitle();
	}
}

// Handle incoming messages for other chat rooms
function handleOtherChatRoomMessages(message) {
	console.log("Message received for another chat room:", message);
	playNotificationSound();
	if (document.hidden) {
		startFlashingTitle();
	}
}

// Function to start flashing the title
function startFlashingTitle() {
	if (!titleInterval) {
		let showNewMessage = true;
		titleInterval = setInterval(() => {
			document.title = showNewMessage ? "New Message" : "WebSocket Chat";
			showNewMessage = !showNewMessage;
		}, 1000);
	}
}

// Reset title and stop flashing when the tab becomes visible
document.addEventListener("visibilitychange", function () {
	if (document.visibilityState === "visible") {
		clearInterval(titleInterval);
		titleInterval = null;
		document.title = "WebSocket Chat";
	}
});

function showNotification(title, message) {
	if (Notification.permission === "granted") {
		new Notification(title, {
			body: message,
			icon: "assets/profile.jpg",
		});
		playNotificationSound();
	} else if (Notification.permission !== "denied") {
		Notification.requestPermission().then((permission) => {
			if (permission === "granted") {
				showNotification(title, message);
			}
		});
	}
}

function playNotificationSound() {
	notificationSound
		.play()
		.catch((error) => console.error("Error playing sound:", error));
}

// Fetch chat rooms from server
function fetchChatRooms() {
	fetch("/api/chatrooms", {
		method: "GET",
		credentials: "include", // Ensure cookies are sent with the request
	})
		.then((response) => response.json())
		.then(handleChatRooms)
		.catch((error) => console.error("Error fetching chat rooms:", error));
}

// Handle chat room selection
function selectChatRoom(chatRoomID) {
	currentChatRoomID = chatRoomID;
	fetchMessageHistory(chatRoomID);
}

// Fetch message history for a chat room
function fetchMessageHistory(chatRoomID) {
	fetch(`/messages/${chatRoomID}`, {
		method: "GET",
		credentials: "include", // Ensure cookies are sent with the request
	})
		.then((response) => response.json())
		.then((messages) => {
			console.log("Fetched messages:");
			if (Array.isArray(messages)) {
				chatBox.innerHTML = "";
				messages.forEach(appendMessageToChatBox);
				adjustScrollPosition(messages);
			} else {
				chatBox.innerHTML = "";
				console.error("Expected an array of messages, but received:", messages);
			}
		})
		.catch((error) => console.error("Error fetching message history:", error));
}

// Handle message deletion
function deleteMessage(messageID, chatRoomID) {
	fetch(`/messages/${messageID}?chat_room_id=${chatRoomID}&user_id=${userID}`, {
		method: "DELETE",
		credentials: "include", // Ensure cookies are sent with the request
	})
		.then((response) => {
			if (!response.ok) {
				throw new Error(`Failed to delete message: ${response.statusText}`);
			}
			console.log("Message deleted successfully");
		})
		.catch((error) => console.error("Error deleting message:", error));
}

// Send a message to the server
document
	.getElementById("message-form")
	.addEventListener("submit", function (e) {
		e.preventDefault();
		const messageInput = document.getElementById("message-input");
		const message = messageInput.value.trim();

		if (message && userID !== undefined && currentChatRoomID !== null) {
			const msg = {
				sender_id: userID,
				chat_room_id: currentChatRoomID,
				content: message,
				is_dm: false,
				timestamp: new Date().toISOString(),
			};
			console.log("Sending message:", msg);
			socket.send(JSON.stringify(msg));
			messageInput.value = ""; // Clear input after sending
		}
	});

// Set to keep track of sent read receipts
const sentReadReceipts = new Set();

// Function to send a read receipt for a specific message
function sendReadReceipt(messageID) {
	if (userID !== undefined && currentChatRoomID !== null) {
		const readReceipt = {
			message_id: parseInt(messageID, 10), // Ensure it's an integer
			chat_room_id: currentChatRoomID,
		};
		// Check if receipt has already been sent
		const receiptKey = `${readReceipt.message_id}-${readReceipt.chat_room_id}`;
		if (!sentReadReceipts.has(receiptKey)) {
			console.log("Sending read receipt:", readReceipt);
			socket.send(JSON.stringify(readReceipt));
			sentReadReceipts.add(receiptKey); // Mark receipt as sent
		}
	}
}

// Check if an element is in the viewport
function isElementInViewport(el) {
	const chatBoxRect = chatBox.getBoundingClientRect();
	const elRect = el.getBoundingClientRect();

	// Check if the element is within the chat box bounds
	const isInChatBox =
		elRect.top < chatBoxRect.bottom && elRect.bottom > chatBoxRect.top;
	return isInChatBox;
}

// Check for read receipts
function checkReadReceipts() {
	const messageElements = Array.from(chatBox.getElementsByClassName("message"));

	messageElements.forEach((messageEl) => {
		if (isElementInViewport(messageEl)) {
			const messageID = messageEl.dataset.messageId;
			sendReadReceipt(messageID);
		}
	});
}

// Set up a periodic check for read receipts
setInterval(checkReadReceipts, 5000);

// Append a message to the chat box
function appendMessageToChatBox(message) {
    const messageElement = document.createElement("div");
    messageElement.className = "message";
    messageElement.dataset.messageId = message.message_id;
    messageElement.dataset.readAt = message.read_at;

    if (message.type === "media") {
        const mediaElement = document.createElement("img");
        mediaElement.src = message.content;
        mediaElement.alt = "Media Image";
        mediaElement.style.maxWidth = "300px";
        mediaElement.style.cursor = "pointer";
        mediaElement.onclick = function() {
            window.open(message.content, '_blank');
        };
        messageElement.appendChild(mediaElement);
    } else {
        messageElement.textContent = `${message.sender.name}: ${message.content}`;
    }

    if (message.read_at !== "1970-01-01T00:00:00Z") {
        messageElement.style.backgroundColor = "#e0ffe0";
    }

    if (message.sender_id === userID) {
        const deleteButton = document.createElement("button");
        deleteButton.textContent = "Delete";
        deleteButton.className = "delete-button";
        deleteButton.addEventListener("click", () =>
            deleteMessage(message.message_id, message.chat_room_id)
        );
        messageElement.appendChild(deleteButton);
    }

    chatBox.appendChild(messageElement);
    chatBox.scrollTop = chatBox.scrollHeight;

    console.log("Appended message to chat box:", message);
}

// Handle scroll events to send read receipts for visible messages
chatBox.addEventListener("scroll", function () {
	checkReadReceipts();
});

function adjustScrollPosition(messages) {
	// Find the index of the last read message
	const lastReadMessageIndex = messages
		.slice()
		.reverse()
		.findIndex((msg) => msg.read_at !== "1970-01-01T00:00:00Z");
	const adjustedLastReadIndex =
		lastReadMessageIndex !== -1
			? messages.length - 1 - lastReadMessageIndex
			: -1;

	// Find the index of the first unread message
	const firstUnreadMessageIndex = messages.findIndex(
		(msg) => msg.read_at === "1970-01-01T00:00:00Z"
	);

	if (
		messages.length === 0 ||
		chatBox.scrollTop + chatBox.clientHeight >= chatBox.scrollHeight
	) {
		// If no messages or already scrolled to the bottom, scroll to the bottom
		chatBox.scrollTop = chatBox.scrollHeight;
	} else if (adjustedLastReadIndex !== -1 && firstUnreadMessageIndex !== -1) {
		// Calculate the midpoint between the last read and first unread message
		const midpointIndex = Math.floor(
			(adjustedLastReadIndex + firstUnreadMessageIndex) / 2
		);
		const element = Array.from(chatBox.children)[midpointIndex];

		if (element) {
			chatBox.scrollTop = element.offsetTop - chatBox.clientHeight / 2;
		}
	} else if (firstUnreadMessageIndex !== -1) {
		// Scroll to the first unread message if present
		const element = Array.from(chatBox.children)[firstUnreadMessageIndex];
		if (element) {
			console.log("Scroll to first unread message");
			chatBox.scrollTop = element.offsetTop - chatBox.clientHeight / 2;
		}
	} else {
		// If all messages are read, scroll to the bottom
		chatBox.scrollTop = chatBox.scrollHeight;
	}
}