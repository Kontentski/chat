
function leaveChatRoom(chatRoomID, chatRoomName) {
    const popup = document.getElementById('leave-group-popup');
    const leaveText = document.getElementById('leave-group-text');
    const confirmButton = document.getElementById('confirm-leave');
    const cancelButton = document.getElementById('cancel-leave');
    
    // Set popup text to include the name of the group
    leaveText.textContent = `Do you really want to leave the group "${chatRoomName}"?`;

    // Show the popup
    popup.style.display = 'flex';

    // Confirm leave group action
    confirmButton.onclick = function() {
        console.log(`User confirmed to leave chat room with ID: ${chatRoomID}`);

        fetch(`/api/chatrooms/leave/${chatRoomID}`, {
            method: 'POST',
            credentials: `include`
        })
        .then(response => {
            if (!response.ok) {
                throw new Error(`Network response was not ok`);
            }
            return response.json
        })
        .then(data => {
            console.log('User left chat room:', data);
            popup.style.display = 'none';
            fetchChatRooms();
        })
        .catch(error => {
            console.error('Error leaving chat room:', error);
        });
    };

    // Cancel leave group action
    cancelButton.onclick = function() {
        popup.style.display = 'none'; // Close the popup without leaving the group
    };
}

// Show Add Members popup
function addMembersToGroup(chatRoomID) {
    const popup = document.getElementById('add-members-popup');
    const closeBtn = document.querySelector('.close-btn');
    const searchBar = document.getElementById('search-bar');
    const userSearchResults = document.getElementById('user-search-results');

    // Clear previous search results
    userSearchResults.innerHTML = '';

    // Show the popup
    popup.style.display = 'flex';

    // Close the popup when clicking the close button
    closeBtn.onclick = function() {
        popup.style.display = 'none';
        searchBar.value = ''; // Clear search bar when closing
    };

    // Close the popup when clicking outside of it
    window.onclick = function(event) {
        if (event.target == popup) {
            popup.style.display = 'none';
            searchBar.value = ''; // Clear search bar when closing
        }
    };

    // Search for users as you type
    searchBar.addEventListener('input', function() {
        const searchQuery = searchBar.value.toLowerCase();

        // Fetch users from server or filter existing list (if available)
        fetch(`/api/chatrooms/search-users?q=${searchQuery}`)
            .then(response => response.json())
            .then(users => {
                // Clear previous results
                userSearchResults.innerHTML = '';

                // Show matching users
                users.forEach(user => {
                    const listItem = document.createElement('li');
                    listItem.textContent = `${user.username} (${user.name})`;

                    // Add click event to add the user to the group
                    listItem.addEventListener('click', () => {
                        addUserToChatRoom(user.user_id, chatRoomID);
                    });

                    userSearchResults.appendChild(listItem);
                });
            })
            .catch(error => console.error('Error fetching users:', error));
    });
}

// Add the selected user to the chat room
function addUserToChatRoom(userID, chatRoomID) {
    fetch('/api/chatrooms/add-user', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            user_id: userID,
            chat_room_id: chatRoomID
        }),
    })
    .then(response => response.json())
    .then(data => {
        if (data.error) {
            alert(`Error: ${data.error}`);
        } else {
            alert('User added to the chat room');
            // Close popup
            document.getElementById('add-members-popup').style.display = 'none';
        }
    })
    .catch(error => console.error('Error adding user to chat room:', error));
}
