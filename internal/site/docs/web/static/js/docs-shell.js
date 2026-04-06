(function () {
	var root = document.documentElement;
	var themeStorageKey = "ui8kit-theme";

	function readStoredTheme() {
		try {
			return localStorage.getItem(themeStorageKey);
		} catch (_) {
			return "";
		}
	}

	function writeStoredTheme(value) {
		try {
			localStorage.setItem(themeStorageKey, value);
		} catch (_) {}
	}

	function resolvePreferredTheme() {
		var storedTheme = readStoredTheme();
		if (storedTheme === "dark" || storedTheme === "light") {
			return storedTheme;
		}

		if (
			window.matchMedia &&
			window.matchMedia("(prefers-color-scheme: dark)").matches
		) {
			return "dark";
		}

		return "light";
	}

	function applyTheme(theme) {
		root.classList.toggle("dark", theme === "dark");
	}

	function applyThemeButtonState() {
		var icon = document.getElementById("theme-toggle-icon");
		var button = document.getElementById("ui8kit-theme-toggle");
		var dark = root.classList.contains("dark");

		if (icon) {
			icon.className = dark
				? "latty latty-sun h-4 w-4"
				: "latty latty-moon h-4 w-4";
		}

		if (button) {
			var switchTo = dark ? button.dataset.switchToLightLabel : button.dataset.switchToDarkLabel;
			button.setAttribute("aria-pressed", dark ? "true" : "false");
			button.setAttribute("title", switchTo || "Toggle theme");
			button.setAttribute("aria-label", switchTo || "Toggle theme");
		}
	}

	applyTheme(resolvePreferredTheme());

	document.addEventListener("DOMContentLoaded", function () {
		var themeButton = document.getElementById("ui8kit-theme-toggle");
		if (!themeButton) {
			return;
		}

		themeButton.addEventListener("click", function () {
			var nextTheme = root.classList.contains("dark") ? "light" : "dark";
			applyTheme(nextTheme);
			writeStoredTheme(nextTheme);
			applyThemeButtonState();
		});

		applyThemeButtonState();
	});
})();
