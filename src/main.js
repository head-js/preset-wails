const statusEl = document.getElementById("status");
const versionEl = document.getElementById("version");

function showError(message) {
	statusEl.textContent = "Failed to load version";
	statusEl.classList.add("error");
	versionEl.hidden = true;
	console.error(message);
}

async function loadVersion() {
	try {
		const response = await fetch("/api/version", { method: "GET" });
		if (!response.ok) {
			showError(`HTTP ${response.status}`);
			return;
		}
		const contentType = response.headers.get("Content-Type") || "";
		if (!contentType.includes("application/json")) {
			showError(`Unexpected content-type: ${contentType}`);
			return;
		}
		const data = await response.json();
		if (typeof data.version !== "string") {
			showError("Response missing version field");
			return;
		}
		statusEl.textContent = "App version:";
		versionEl.textContent = data.version;
		versionEl.hidden = false;
	} catch (err) {
		showError(err);
	}
}

loadVersion();