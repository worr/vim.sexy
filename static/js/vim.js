"use strict";
var vim = (function() {
	var obj = {};
	var lock = false;

	var clearFields = function() {
		document.getElementById("email").value = "";
		document.getElementsByName("csrf_token")[0].value = "";
		Recaptcha.reload();
	}

	var validateEmail = function() {
		return document.getElementById("email").value !== "";
	};

	var fade = function(ele, opacity, fn) {
		ele.style.opacity = opacity;
		setTimeout(function() { fn(ele, opacity); }, 25);
	}

	var fadeIn = function(ele, opacity) {
		if (isNaN(opacity)) opacity = 0.0;
		opacity += 0.05;

		if (window.getComputedStyle(ele).opacity !== "1") {
			fade(ele, opacity, fadeIn);
		} else {
			setTimeout(function() { fadeOut(ele); }, 5000);
		}
	};

	var fadeOut = function(ele, opacity) {
		if (isNaN(opacity)) opacity = 1.0;
		opacity -= 0.05;

		if (window.getComputedStyle(ele).opacity !== "0") {
			fade(ele, opacity, fadeOut);
		} else {
			lock = false;
		}
	}

	var flash = function(isError, message) {
		var color = "#0c0";
		if (isError) color = "#c00";

		var flash = document.getElementById("flash");
		flash.style.background = color;
		flash.textContent = message;
		fadeIn(flash);
	};

	var errorFlash = function(error) {
		if (lock) return;
		lock = true;
		flash(true, error);
	};

	var successFlash = function(message) {
		if (lock) return;
		lock = true;
		flash(false, message);
	};

	// Set up our listeners
	document.addEventListener("DOMContentLoaded", function() {
		document.getElementById("email-form").addEventListener("submit", function(e) {
			e.preventDefault();
			e.stopPropagation();
			if (!validateEmail()) {
				errorFlash("Must provide email address");
				return false;
			}

			var data = {};
			data["email"] = document.getElementById("email").value;
			data["csrf_token"] = document.getElementsByName("csrf_token")[0].value;
			data["recaptcha_challenge_field"] = document.getElementsByName("recaptcha_challenge_field")[0].value;
			data["recaptcha_response_field"] = document.getElementsByName("recaptcha_response_field")[0].value;

			var req = new XMLHttpRequest();
			req.open("POST", "/", true);
			req.onreadystatechange = function() {
				if (req.readyState !== 4 || req.status != 200) {
					if (req.responseText) {
						try {
							errorFlash(req.responseText);
						} catch(e) {
							errorFlash(req.responseText);
						}
					} else {
						errorFlash(req.statusText);
					}
				} else {
					successFlash("Invite sent");
				}

				clearFields();
			};
			req.setRequestHeader("Content-type", "application/x-www-form-urlencoded");
			req.send("email=" + encodeURIComponent(data["email"]) +
					 "&csrf_token=" + encodeURIComponent(data["csrf_token"]) +
					 "&recaptcha_challenge_field=" + encodeURIComponent(data["recaptcha_challenge_field"]) +
					 "&recaptcha_response_field=" + encodeURIComponent(data["recaptcha_response_field"]));

			return false;
		});
	});

	return obj;
})();
