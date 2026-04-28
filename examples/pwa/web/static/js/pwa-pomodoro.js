(function () {
  const SETTINGS_KEY = "fastygo-pwa:pomodoro-settings";
  const MOBILE_QUERY = "(max-width: 760px)";
  const SECOND = 1000;

  const defaults = {
    work: 25,
    shortBreak: 5,
    longBreak: 20,
    round: 4,
    goal: 12,
    sound: true,
    tick: false,
    notifications: false
  };

  document.addEventListener("DOMContentLoaded", function () {
    const root = document.querySelector("[data-pwa-pomodoro]");
    if (!root || window.matchMedia(MOBILE_QUERY).matches) {
      return;
    }

    createPomodoro(root);
  });

  function createPomodoro(root) {
    const card = root.querySelector(".pwa-pomodoro-card");
    const settingsForm = root.querySelector("[data-pomodoro-settings]");
    const startButton = root.querySelector("[data-pomodoro-start]");
    const skipButton = root.querySelector("[data-pomodoro-skip]");
    const resetButton = root.querySelector("[data-pomodoro-reset]");
    const modeLabel = root.querySelector("[data-pomodoro-mode]");
    const timeLabel = root.querySelector("[data-pomodoro-time]");
    const progress = root.querySelector("[data-pomodoro-progress]");
    const goalLabel = root.querySelector("[data-pomodoro-goal]");
    const originalTitle = document.title;

    if (!card || !settingsForm || !startButton || !skipButton || !resetButton || !modeLabel || !timeLabel || !progress || !goalLabel) {
      return;
    }

    let settings = readSettings();
    let mode = "work";
    let completed = 0;
    let running = false;
    let endAt = 0;
    let remainingMs = durationForMode(mode, settings);
    let interval = 0;
    let lastSecond = null;
    let audioContext = null;

    hydrateSettings(settingsForm, settings);
    render();

    startButton.addEventListener("click", function () {
      unlockAudio();
      if (running) {
        pause();
      } else {
        start();
      }
    });

    skipButton.addEventListener("click", function () {
      unlockAudio();
      finishPhase();
    });

    resetButton.addEventListener("click", function () {
      reset();
    });

    settingsForm.addEventListener("input", function () {
      settings = readSettingsFromForm(settingsForm);
      writeSettings(settings);
      if (!running) {
        remainingMs = durationForMode(mode, settings);
      }
      if (settings.notifications) {
        requestNotifications();
      }
      render();
    });

    window.addEventListener("focus", render);
    document.addEventListener("visibilitychange", render);

    function start() {
      running = true;
      endAt = Date.now() + remainingMs;
      lastSecond = null;
      window.clearInterval(interval);
      interval = window.setInterval(tick, 250);
      tick();
    }

    function pause() {
      remainingMs = Math.max(0, endAt - Date.now());
      running = false;
      window.clearInterval(interval);
      render();
    }

    function reset() {
      running = false;
      window.clearInterval(interval);
      mode = "work";
      completed = 0;
      remainingMs = durationForMode(mode, settings);
      document.title = originalTitle;
      render();
    }

    function tick() {
      remainingMs = Math.max(0, endAt - Date.now());
      maybeTickSound();
      if (remainingMs <= 0) {
        finishPhase();
        return;
      }
      render();
    }

    function finishPhase() {
      const finishedMode = mode;
      if (finishedMode === "work") {
        completed += 1;
      }

      playAlarm();
      notify(finishedMode);
      mode = nextMode(finishedMode, completed, settings);
      remainingMs = durationForMode(mode, settings);
      running = false;
      window.clearInterval(interval);
      render();
      start();
    }

    function render() {
      if (running) {
        remainingMs = Math.max(0, endAt - Date.now());
      }

      const duration = durationForMode(mode, settings);
      const elapsed = Math.max(0, duration - remainingMs);
      const progressValue = duration > 0 ? Math.min(100, (elapsed / duration) * 100) : 0;
      const modeText = modeName(card, mode);
      const timeText = formatTime(remainingMs);

      modeLabel.textContent = modeText;
      timeLabel.textContent = timeText;
      progress.style.width = progressValue + "%";
      goalLabel.textContent = formatGoal(goalLabel.textContent, completed, settings.goal);
      startButton.textContent = running ? card.getAttribute("data-pause-label") : card.getAttribute("data-start-label");
      root.classList.toggle("pwa-pomodoro-running", running);
      root.classList.toggle("pwa-pomodoro-resting", mode !== "work");
      document.title = timeText + " · " + modeText + " — " + originalTitle;
    }

    function maybeTickSound() {
      if (!settings.tick || !audioContext) {
        return;
      }

      const currentSecond = Math.ceil(remainingMs / SECOND);
      if (currentSecond !== lastSecond && lastSecond !== null) {
        playTone(520, 0.025, 0.025);
      }
      lastSecond = currentSecond;
    }

    function playAlarm() {
      if (!settings.sound || !audioContext) {
        return;
      }

      playTone(660, 0.14, 0.08);
      window.setTimeout(function () { playTone(880, 0.16, 0.08); }, 180);
    }

    function unlockAudio() {
      if (audioContext) {
        if (audioContext.state === "suspended") {
          audioContext.resume();
        }
        return;
      }

      const Context = window.AudioContext || window.webkitAudioContext;
      if (!Context) {
        return;
      }

      audioContext = new Context();
      if (audioContext.state === "suspended") {
        audioContext.resume();
      }
    }

    function playTone(frequency, duration, volume) {
      const oscillator = audioContext.createOscillator();
      const gain = audioContext.createGain();
      const now = audioContext.currentTime;

      oscillator.type = "sine";
      oscillator.frequency.setValueAtTime(frequency, now);
      gain.gain.setValueAtTime(0.0001, now);
      gain.gain.exponentialRampToValueAtTime(volume, now + 0.01);
      gain.gain.exponentialRampToValueAtTime(0.0001, now + duration);
      oscillator.connect(gain);
      gain.connect(audioContext.destination);
      oscillator.start(now);
      oscillator.stop(now + duration + 0.03);
    }

    function notify(finishedMode) {
      if (!settings.notifications || !("Notification" in window) || Notification.permission !== "granted") {
        return;
      }

      const next = nextMode(finishedMode, completed, settings);
      new Notification(modeName(card, next), {
        body: modeName(card, finishedMode) + " complete"
      });
    }
  }

  function readSettings() {
    try {
      const parsed = JSON.parse(sessionStorage.getItem(SETTINGS_KEY) || "{}");
      return normalizeSettings(Object.assign({}, defaults, parsed));
    } catch (_) {
      return Object.assign({}, defaults);
    }
  }

  function writeSettings(settings) {
    sessionStorage.setItem(SETTINGS_KEY, JSON.stringify(normalizeSettings(settings)));
  }

  function hydrateSettings(form, settings) {
    form.elements.work.value = settings.work;
    form.elements.shortBreak.value = settings.shortBreak;
    form.elements.longBreak.value = settings.longBreak;
    form.elements.round.value = settings.round;
    form.elements.goal.value = settings.goal;
    form.elements.sound.checked = settings.sound;
    form.elements.tick.checked = settings.tick;
    form.elements.notifications.checked = settings.notifications;
  }

  function readSettingsFromForm(form) {
    return normalizeSettings({
      work: form.elements.work.value,
      shortBreak: form.elements.shortBreak.value,
      longBreak: form.elements.longBreak.value,
      round: form.elements.round.value,
      goal: form.elements.goal.value,
      sound: form.elements.sound.checked,
      tick: form.elements.tick.checked,
      notifications: form.elements.notifications.checked
    });
  }

  function normalizeSettings(settings) {
    return {
      work: clamp(number(settings.work), 5, 55),
      shortBreak: clamp(number(settings.shortBreak), 3, 10),
      longBreak: clamp(number(settings.longBreak), 5, 45),
      round: clamp(number(settings.round), 1, 12),
      goal: clamp(number(settings.goal), 1, 24),
      sound: Boolean(settings.sound),
      tick: Boolean(settings.tick),
      notifications: Boolean(settings.notifications)
    };
  }

  function durationForMode(mode, settings) {
    if (mode === "shortBreak") {
      return settings.shortBreak * 60 * SECOND;
    }
    if (mode === "longBreak") {
      return settings.longBreak * 60 * SECOND;
    }
    return settings.work * 60 * SECOND;
  }

  function nextMode(mode, completed, settings) {
    if (mode !== "work") {
      return "work";
    }
    return completed > 0 && completed % settings.round === 0 ? "longBreak" : "shortBreak";
  }

  function modeName(card, mode) {
    if (mode === "shortBreak") {
      return card.getAttribute("data-mode-short-break");
    }
    if (mode === "longBreak") {
      return card.getAttribute("data-mode-long-break");
    }
    return card.getAttribute("data-mode-work");
  }

  function formatTime(ms) {
    const total = Math.max(0, Math.ceil(ms / SECOND));
    const minutes = Math.floor(total / 60);
    const seconds = total % 60;
    return pad(minutes) + ":" + pad(seconds);
  }

  function formatGoal(template, completed, goal) {
    if (template.includes(" из ")) {
      return completed + " из " + goal + " помидорок";
    }
    return completed + " of " + goal + " pomodoros";
  }

  function requestNotifications() {
    if ("Notification" in window && Notification.permission === "default") {
      Notification.requestPermission();
    }
  }

  function number(value) {
    const parsed = Number.parseInt(value, 10);
    return Number.isFinite(parsed) ? parsed : 0;
  }

  function clamp(value, min, max) {
    return Math.min(max, Math.max(min, value));
  }

  function pad(value) {
    return String(value).padStart(2, "0");
  }
})();
