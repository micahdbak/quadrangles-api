async function submit() {
	const message = document.getElementById("message");

	const fileForm = new FormData();
	const fileInput = document.getElementById("file");

	fileForm.append("file", fileInput.files[0]);

	let file;

	try {
		file = await fetch("/api/file", {
			method: "POST",
			mode: "no-cors",
			headers: {
				"Content-Type": "multipart/form"
			},
			body: fileForm
		});
	} catch {
		message.innerHTML = "Failed to upload file.";
		return;
	}

	if (!file.ok) {
		message.innerHTML = "File response not OK.";
		return;
	}

	const fid = parseInt(await file.text());
	const topic = document.getElementById("topic").value;
	const text = document.getElementById("text").value;

	message.innerHTML = `fid: ${fid}, topic: ${topic}, text: ${text}`;

	const postForm = new FormData();

	postForm.append("fid", fid);
	postForm.append("topic", topic);
	postForm.append("text", text);

	let post;

	try {
		post = await fetch("/api/post", {
			method: "POST",
			mode: "no-cors",
			body: postForm
		});
	} catch {
		message.innerHTML = "Failed to create post.";
		return;
	}

	if (!post.ok) {
		message.innerHTML = "Post response not OK.";
		return;
	}

	const pid = await post.json();

	window.location = `/api/p/${pid}`;
}
