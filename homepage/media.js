async function uploadFile(file) {
	const formData = new FormData();

	// Include the chatRoomID in the form data
	formData.append("chat_room_id", currentChatRoomID);
	formData.append("sender_id", userID);
	formData.append("file", file);
	// Send file to the server using fetch API
	const response = await fetch("/api/upload-media", {
		method: "POST",
		body: formData,
	});

	const result = await response.json();


	// Send the media URL through WebSocket to broadcast the message
	socket.send(
		JSON.stringify({
			type: "media",
			content: result.filePath,
			sender_id: userID,
			chat_room_id: currentChatRoomID,
		})
	);
}
