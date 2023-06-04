const ws = new WebSocket("ws://localhost:8000/api/ws/1");

function addMessage(m) {
	message = document.createElement("p");
	message.innerHTML = m;

	messages = document.getElementById("messages");
	messages.appendChild(message);
}

function sendMessage() {
	message = document.getElementById("message");

	ws.send(message.value);
}

window.onload = () => {
	ws.addEventListener("message", (e) => {
		addMessage(e.data);
	});
};
