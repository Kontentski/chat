let userID;
let socket;
let currentChatRoomID = null;
let pingInterval = 60000; // 60 seconds
let titleInterval = null;

const notificationSound = new Audio('assets/notification.mp3');
const username = prompt('Enter your username:');
const chatRoomList = document.getElementById('chat-room-list-items');
const chatBox = document.getElementById('chat-box');

// Connect to WebSocket server
function connectWebSocket() {
    socket = new WebSocket(`wss://electric-wasp-cool.ngrok-free.app/ws?username=${username}`);

    socket.onopen = function() {
        console.log('Connected to WebSocket server');
    };

    socket.onmessage = function(event) {
        const data = JSON.parse(event.data);

        if (data.token) {
            handleToken(data.token);
        } else if (data.userID) {
            handleUserID(data);
        } else if (Array.isArray(data)) {
            handleChatRooms(data);
        } else if (data.chat_room_id === currentChatRoomID) {
            handleIncomingMessage(data);
        } else {
            handleOtherChatRoomMessages(data);
        }
    };

    socket.onclose = function(event) {
        console.log('WebSocket connection closed', event);
        clearInterval(pingInterval); // Clear ping interval
        reconnectWebSocket(); // Attempt to reconnect
    };

    socket.onerror = function(error) {
        console.error('WebSocket error:', error);
    };
}


function reconnectWebSocket() {
    console.log('Attempting to reconnect WebSocket...');
    setTimeout(() => {
        connectWebSocket();
    }, 5000); // Retry after 5 seconds
}

connectWebSocket();

// Handle token reception
function handleToken(token) {
    localStorage.setItem('token', token);
    console.log('Token received and stored');
}

// Handle user ID reception
function handleUserID(data) {
    userID = data.userID;
    localStorage.setItem('userID', userID);
    document.getElementById('user-username').textContent = `Username: ${username}`;
    document.getElementById('user-name').textContent = `Name: ${data.name}`;
    fetchChatRooms();
}

// Handle chat rooms data
function handleChatRooms(chatRooms) {
    chatRoomList.innerHTML = '';
    chatRooms.forEach(room => {
        const listItem = document.createElement('li');
        listItem.textContent = room.name;
        listItem.dataset.chatRoomId = room.id;
        listItem.addEventListener('click', () => selectChatRoom(room.id));
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
    console.log('Message received for another chat room:', message);
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
            document.title = showNewMessage ? 'New Message' : 'WebSocket Chat';
            showNewMessage = !showNewMessage;
        }, 1000);
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

function showNotification(title, message) {
    if (Notification.permission === "granted") {
        new Notification(title, {
            body: message,
            icon: 'assets/profile.jpg',
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
    .then(handleChatRooms)
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
    fetch(`/messages/${chatRoomID}?userID=${userID}`, {
        method: 'GET',
        headers: {
            'Authorization': `Bearer ${token}`
        }
    })
    .then(response => {
        if (!response.ok) {
            throw new Error(`Network response was not ok ${response.statusText}`);
        }
        return response.json();
    })
    .then(messages => {
        if (Array.isArray(messages)) {
            chatBox.innerHTML = '';
            messages.forEach(appendMessageToChatBox);
            adjustScrollPosition(messages);
        } else {
            chatBox.innerHTML = '';
            console.error('Expected an array of messages, but received:', messages);
        }
    })
    .catch(error => console.error('Error fetching message history:', error));
}

// Send a message to the server
document.getElementById('message-form').addEventListener('submit', function(e) {
    e.preventDefault();
    const messageInput = document.getElementById('message-input');
    const message = messageInput.value.trim();

    if (message && userID !== undefined && currentChatRoomID !== null) {
        const msg = {
            sender_id: userID,
            chat_room_id: currentChatRoomID,
            content: message,
            is_dm: false,
            timestamp: new Date().toISOString()
        };
        console.log('Sending message:', msg);
        socket.send(JSON.stringify(msg));
        messageInput.value = ''; // Clear input after sending
    }
});

// Set to keep track of sent read receipts
const sentReadReceipts = new Set();

// Function to send a read receipt for a specific message
function sendReadReceipt(messageID) {
    console.log('Sending read receipt with messageID:', messageID);
    if (userID !== undefined && currentChatRoomID !== null) {
        const readReceipt = {
            message_id: parseInt(messageID, 10),  // Ensure it's an integer
            chat_room_id: currentChatRoomID
        };
        // Check if receipt has already been sent
        const receiptKey = `${readReceipt.message_id}-${readReceipt.chat_room_id}`;
        if (!sentReadReceipts.has(receiptKey)) {
            console.log('Sending read receipt:', readReceipt);
            socket.send(JSON.stringify(readReceipt));
            sentReadReceipts.add(receiptKey); // Mark receipt as sent
        }
    }
}



function isElementInViewport(el) {
    const rect = el.getBoundingClientRect();
    return (
        rect.top >= 0 &&
        rect.left >= 0 &&
        rect.bottom <= (window.innerHeight || document.documentElement.clientHeight) &&
        rect.right <= (window.innerWidth || document.documentElement.clientWidth)
    );
}

function checkReadReceipts() {
    const messageElements = Array.from(chatBox.children);
    messageElements.forEach(messageElement => {
        if (isElementInViewport(messageElement)) {
            const messageID = messageElement.dataset.messageId;
            const readAt = messageElement.dataset.readAt;
            if (messageID && (readAt === '1970-01-01T00:00:00Z' || readAt === null)) {
                sendReadReceipt(messageID);
            }
        }
    });
}


// Call sendReadReceipt() when a message is read
function appendMessageToChatBox(message) {
    const messageElement = document.createElement('div');
    messageElement.textContent = `${message.sender.name}: ${message.content}`;

    if (message.read_at !== '1970-01-01T00:00:00Z') {
        messageElement.style.backgroundColor = '#e0ffe0'; // Highlight read messages
    }

    messageElement.dataset.messageId = message.message_id;
    messageElement.dataset.readAt = message.read_at;  
    chatBox.appendChild(messageElement);

    console.log('Appending message to chat box:', message);

    // Check if the message is visible and send read receipt
    if (isElementInViewport(messageElement)) {
        if (message.read_at === '1970-01-01T00:00:00Z' || message.read_at === null) {
            sendReadReceipt(message.message_id);
        }
    }
}

// Handle scroll events to send read receipts for visible messages
chatBox.addEventListener('scroll', function() {
    checkReadReceipts();
});

// Adjust scroll position based on read messages
function adjustScrollPosition(messages) {
    // Find the index of the first unread message
    const firstUnreadMessageIndex = messages.findIndex(msg => msg.read_at === '1970-01-01T00:00:00Z');
    
    if (firstUnreadMessageIndex !== -1) {
        // Scroll to the first unread message
        const element = Array.from(chatBox.children)[firstUnreadMessageIndex];
        if (element) {
            chatBox.scrollTop = element.offsetTop;
        }
    } else {
        // If all messages are read, scroll to the bottom
        chatBox.scrollTop = chatBox.scrollHeight;
    }
}

