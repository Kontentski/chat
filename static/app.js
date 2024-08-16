let userID;
let socket;
let currentChatRoomID = null;

const notificationSound = new Audio('assets/notification.mp3');
const username = prompt('Enter your username:');
const chatRoomList = document.getElementById('chat-room-list-items');
const chatBox = document.getElementById('chat-box');

// Connect to WebSocket server
socket = new WebSocket(`ws://localhost:8080/ws?username=${username}`);

socket.onopen = function() {
    console.log('Connected to WebSocket server');
};

let titleInterval;

socket.onmessage = function(event) {
    const data = JSON.parse(event.data);

    if (data.token) {
        // Store token in localStorage
        localStorage.setItem('token', data.token);
        console.log('Token received and stored');
    } else if (data.userID !== undefined) {
        userID = data.userID;
        document.getElementById('user-username').textContent = `Username: ${username}`;
        document.getElementById('user-name').textContent = `Name: ${data.name}`;

        // Fetch chat rooms
        fetchChatRooms();
    } else if (Array.isArray(data)) {
        // Handle chat rooms data
        chatRoomList.innerHTML = '';
        data.forEach(room => {
            const listItem = document.createElement('li');
            listItem.textContent = room.name;
            listItem.dataset.chatRoomId = room.id;
            listItem.addEventListener('click', () => selectChatRoom(room.id));
            chatRoomList.appendChild(listItem);
        });
    } else if (data.chat_room_id === currentChatRoomID) {
        // Handle incoming messages for the current chat room
        appendMessageToChatBox(data);

        // If the tab is not focused, play a notification sound and start flashing title
        if (document.hidden) {
            playNotificationSound();
            startFlashingTitle();
        }
    } else {
        // Handle incoming messages for other chat rooms
        console.log('Message received for another chat room:', data);
        playNotificationSound(); // Play notification sound for messages from other chat rooms

        // If the tab is not focused, start flashing title
        if (document.hidden) {
            startFlashingTitle();
        }
    }
};

// Function to start flashing the title
function startFlashingTitle() {
    if (!titleInterval) {
        let showNewMessage = true;
        titleInterval = setInterval(() => {
            document.title = showNewMessage ? 'New Message' : 'WebSocket Chat';
            showNewMessage = !showNewMessage;
        }, 1000); // Change every second
    }
}

// Reset title and stop flashing when the tab becomes visible
document.addEventListener('visibilitychange', function() {
    if (document.visibilityState === 'visible') {
        clearInterval(titleInterval);
        titleInterval = null;
        document.title = 'WebSocket Chat';
    }
});



socket.onclose = function(event) {
    console.log('WebSocket connection closed', event);
};

socket.onerror = function(error) {
    console.error('WebSocket error:', error);
};

function showNotification(title, message) {
if (Notification.permission === "granted") {
new Notification(title, {
    body: message,
    icon: 'assets/profile.jpg', // Optional: Add an icon
});
playNotificationSound();
} else if (Notification.permission !== "denied") {
Notification.requestPermission().then(permission => {
    if (permission === "granted") {
        showNotification(title, message);
    }
});
}
}

function playNotificationSound() {
    notificationSound.play().catch(error => console.error("Error playing sound:", error));
}

// Fetch chat rooms from server
function fetchChatRooms() {
    const token = localStorage.getItem('token');
    fetch('/api/chatrooms', {
        method: 'GET',
        headers: {
            'Authorization': `Bearer ${token}`
        }
    })
    .then(response => response.json())
    .then(chatRooms => {
        chatRoomList.innerHTML = '';
        chatRooms.forEach(room => {
            const listItem = document.createElement('li');
            listItem.textContent = room.name;
            listItem.dataset.chatRoomId = room.id;
            listItem.addEventListener('click', () => selectChatRoom(room.id));
            chatRoomList.appendChild(listItem);
        });
    })
    .catch(error => console.error('Error fetching chat rooms:', error));
}

// Handle chat room selection
function selectChatRoom(chatRoomID) {
    currentChatRoomID = chatRoomID;
    fetchMessageHistory(chatRoomID);
}

// Fetch message history for a chat room
function fetchMessageHistory(chatRoomID) {
    const token = localStorage.getItem('token');
    fetch(`/messages/${chatRoomID}`, {
        method: 'GET',
        headers: {
            'Authorization': `Bearer ${token}`
        }
    })
    .then(response => response.json())
    .then(messages => {
        chatBox.innerHTML = ''; // Clear existing messages
        messages.forEach(msg => appendMessageToChatBox(msg));
    })
    .catch(error => console.error('Error fetching message history:', error));
}

// Send a message to the server
document.getElementById('message-form').addEventListener('submit', function(e) {
    e.preventDefault();
    const messageInput = document.getElementById('message-input');
    const message = messageInput.value;

    if (message.trim() && userID !== undefined && currentChatRoomID !== null) {
        const msg = {
            sender_id: userID,
            chat_room_id: currentChatRoomID,
            content: message,
            is_dm: false,
            timestamp: new Date().toISOString() // Ensure timestamp format matches server expectation
        };
        console.log('Sending message:', msg);

        socket.send(JSON.stringify(msg));
        messageInput.value = ''; // Clear input after sending
    }
});

function appendMessageToChatBox(message) {
    const messageElement = document.createElement('div');
    messageElement.textContent = `${message.sender.name}: ${message.content}`;
    chatBox.appendChild(messageElement);
    chatBox.scrollTop = chatBox.scrollHeight; // Scroll to bottom
}