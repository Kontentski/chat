async function uploadFile(file) {
	const formData = new FormData();
	formData.append("file", file);

	// Send file to the server using fetch API
	const response = await fetch("/upload-media", {
		method: "POST",
		body: formData,
	});

	const result = await response.json();

	// The server response should contain the uploaded file's URL
	const mediaUrl = result.url;

	// Send the media URL through WebSocket to broadcast the message
	socket.send(
		JSON.stringify({
			type: "media",
			content: mediaUrl,
		})
	);
}

function handleMediaMessage(data) {
	const chatBox = document.getElementById("chat-box");

	if (data.media_url && data.media_type) {
		let mediaElement;

		if (data.type === "image") {
			mediaElement = document.createElement("img");
			mediaElement.src = data.media_url;
			mediaElement.alt = "Media Image";
			mediaElement.style.maxWidth = "300px"; // Limit size
		} else if (data.type === "video") {
			mediaElement = document.createElement("video");
			mediaElement.src = data.media_url;
			mediaElement.controls = true;
			mediaElement.style.maxWidth = "300px";
		} else {
			console.error("Unsupported media type:", data.media_type);
			return;
		}

		chatBox.appendChild(mediaElement);
	} else {
		console.error("Invalid media data received");
	}
}
