(function () {
  const CACHE_PREFIX = "fastygo-pwa-";
  const STORAGE_PREFIX = "fastygo-pwa:";
  const ACTIVE_DB_KEY = STORAGE_PREFIX + "active-db";
  const DB_LIST_KEY = STORAGE_PREFIX + "db-list";
  const SUBSCRIPTION_KEY = STORAGE_PREFIX + "subscription-active";
  const DEFAULT_DB_ID = "default";
  let defaultTaskMessage = "";

  document.addEventListener("DOMContentLoaded", function () {
    defaultTaskMessage = readSubscriptionHint();

    if (document.querySelector("[data-pwa-subscription-success]")) {
      activateSubscription();
    }

    ensureDatabase();
    bindExport();
    bindImport();
    bindClear();
    bindSubscriptionCancel();
    bindTaskForm();
    bindTaskList();
    bindOnboarding();
    applySubscriptionState();
    renderTasks();
  });

  function bindExport() {
    const button = document.querySelector("[data-pwa-export]");
    if (!button) {
      return;
    }

    button.addEventListener("click", function () {
      const database = readActiveDatabase();
      const payload = JSON.stringify(database, null, 2);
      const blob = new Blob([payload], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = database.id + "-tasks.json";
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(url);
      setStatus("Exported " + database.tasks.length + " tasks.");
    });
  }

  function bindImport() {
    const input = document.querySelector("[data-pwa-import]");
    if (!input) {
      return;
    }

    input.addEventListener("change", function () {
      const file = input.files && input.files[0];
      if (!file) {
        return;
      }

      const reader = new FileReader();
      reader.addEventListener("load", function () {
        try {
          const database = normalizeImportedDatabase(JSON.parse(String(reader.result || "{}")));
          writeDatabase(database);
          setActiveDatabase(database.id);
          renderTasks();
          setStatus("Imported " + database.tasks.length + " tasks into " + database.name + ".");
        } catch (error) {
          setStatus("Import failed: " + error.message);
        } finally {
          input.value = "";
        }
      });
      reader.readAsText(file);
    });
  }

  function bindClear() {
    const button = document.querySelector("[data-pwa-clear-cache]");
    if (!button) {
      return;
    }

    button.addEventListener("click", async function () {
      button.disabled = true;
      try {
        await clearPwaData();
        applySubscriptionState();
        renderTasks();
        setStatus("Offline cache, subscription state, and local PWA databases cleared.");
      } catch (error) {
        setStatus("Clear failed: " + error.message);
      } finally {
        button.disabled = false;
      }
    });
  }

  function bindSubscriptionCancel() {
    const link = document.querySelector("[data-pwa-cancel-subscription]");
    if (!link) {
      return;
    }

    link.addEventListener("click", function () {
      deactivateSubscription();
    });
  }

  function bindTaskForm() {
    const form = document.querySelector("[data-pwa-task-form]");
    if (!form) {
      return;
    }

    form.addEventListener("submit", function (event) {
      event.preventDefault();
      if (!isSubscriptionActive()) {
        setTaskMessage(readLockMessage());
        return;
      }

      const formData = new FormData(form);
      const title = String(formData.get("title") || "").trim();
      if (!title) {
        return;
      }

      const database = readActiveDatabase();
      const id = safeID(formData.get("id")) || uniqueTaskID(database.tasks, title);
      const nextTask = {
        id: id,
        title: title,
        time: String(formData.get("time") || "").trim(),
        status: String(formData.get("status") || "Planned").trim() || "Planned",
        priority: String(formData.get("priority") || "").trim()
      };

      const existingIndex = database.tasks.findIndex(function (task) { return task.id === id; });
      if (existingIndex >= 0) {
        database.tasks[existingIndex] = nextTask;
      } else {
        database.tasks.push(nextTask);
      }

      writeDatabase(database);
      resetTaskForm(form);
      renderTasks();
      setTaskMessage("Saved " + nextTask.title + ".");
    });

    form.addEventListener("reset", function () {
      window.setTimeout(function () {
        resetTaskForm(form);
      }, 0);
    });
  }

  function bindTaskList() {
    const list = document.querySelector("[data-pwa-task-list]");
    if (!list) {
      return;
    }

    list.addEventListener("click", function (event) {
      const action = event.target.closest("[data-pwa-task-action]");
      if (!action || !isSubscriptionActive()) {
        return;
      }

      const id = action.getAttribute("data-pwa-task-action-id");
      if (!id) {
        return;
      }

      if (action.getAttribute("data-pwa-task-action") === "edit") {
        loadTaskIntoForm(id);
        return;
      }

      if (action.getAttribute("data-pwa-task-action") === "delete") {
        const database = readActiveDatabase();
        database.tasks = database.tasks.filter(function (task) { return task.id !== id; });
        writeDatabase(database);
        renderTasks();
        setTaskMessage("Deleted task.");
      }
    });
  }

  function bindOnboarding() {
    const container = document.querySelector("[data-pwa-onboarding]");
    const next = document.querySelector("[data-pwa-onboarding-next]");
    if (!container || !next) {
      return;
    }

    const cards = Array.from(container.querySelectorAll("[data-pwa-onboarding-card]"));
    let activeIndex = 0;

    function showActiveCard() {
      cards.forEach(function (card, index) {
        card.classList.toggle("pwa-onboarding-card-active", index === activeIndex);
      });
    }

    next.addEventListener("click", function () {
      if (activeIndex < cards.length - 1) {
        activeIndex += 1;
        showActiveCard();
        return;
      }
      window.location.href = "/paywall";
    });

    showActiveCard();
  }

  function ensureDatabase() {
    const activeID = localStorage.getItem(ACTIVE_DB_KEY) || DEFAULT_DB_ID;
    if (readDatabase(activeID)) {
      return;
    }
    const database = createDatabase(activeID, "Default task database", seedTasksFromDOM());
    writeDatabase(database);
    setActiveDatabase(database.id);
  }

  function createDatabase(id, name, tasks) {
    return {
      schema: "fastygo-pwa.tasks.v1",
      id: safeID(id || DEFAULT_DB_ID),
      name: String(name || "Task database"),
      exportedAt: new Date().toISOString(),
      tasks: normalizeTasks(tasks)
    };
  }

  function readActiveDatabase() {
    ensureDatabase();
    return readDatabase(localStorage.getItem(ACTIVE_DB_KEY) || DEFAULT_DB_ID);
  }

  function readDatabase(id) {
    const raw = localStorage.getItem(databaseKey(id));
    if (!raw) {
      return null;
    }
    try {
      return normalizeDatabase(JSON.parse(raw), { allowEmptyTasks: true });
    } catch (_) {
      return null;
    }
  }

  function writeDatabase(database) {
    const normalized = normalizeDatabase(database, { allowEmptyTasks: true });
    normalized.exportedAt = new Date().toISOString();
    localStorage.setItem(databaseKey(normalized.id), JSON.stringify(normalized));
    rememberDatabase(normalized.id);
  }

  function setActiveDatabase(id) {
    localStorage.setItem(ACTIVE_DB_KEY, safeID(id || DEFAULT_DB_ID));
  }

  function rememberDatabase(id) {
    const list = readDatabaseList();
    if (!list.includes(id)) {
      list.push(id);
    }
    localStorage.setItem(DB_LIST_KEY, JSON.stringify(list));
  }

  function readDatabaseList() {
    try {
      const parsed = JSON.parse(localStorage.getItem(DB_LIST_KEY) || "[]");
      return Array.isArray(parsed) ? parsed.map(safeID).filter(Boolean) : [];
    } catch (_) {
      return [];
    }
  }

  function normalizeImportedDatabase(input) {
    return normalizeDatabase(input, { allowEmptyTasks: false });
  }

  function normalizeDatabase(input, options) {
    const source = input && typeof input === "object" ? input : {};
    const database = source.database && typeof source.database === "object" ? source.database : {};
    const id = safeID(source.id || database.id || DEFAULT_DB_ID);
    const name = String(source.name || database.name || "Imported task database");
    const tasks = normalizeTasks(source.tasks);
    if (tasks.length === 0 && !options.allowEmptyTasks) {
      throw new Error("JSON must contain a non-empty tasks array.");
    }
    return createDatabase(id, name, tasks);
  }

  function normalizeTasks(tasks) {
    if (!Array.isArray(tasks)) {
      return [];
    }
    return tasks
      .map(function (task, index) {
        const title = String(task && task.title || "").trim();
        if (!title) {
          return null;
        }
        return {
          id: safeID(task.id || title) || "task-" + index,
          title: title,
          time: String(task.time || ""),
          status: String(task.status || "Planned"),
          priority: String(task.priority || "")
        };
      })
      .filter(Boolean);
  }

  function seedTasksFromDOM() {
    const tasks = Array.from(document.querySelectorAll("[data-pwa-task]")).map(function (node, index) {
      return {
        id: safeID(node.getAttribute("data-pwa-task-title") || "task-" + index),
        title: node.getAttribute("data-pwa-task-title") || "Task " + (index + 1),
        time: node.getAttribute("data-pwa-task-time") || "",
        status: node.getAttribute("data-pwa-task-status") || "Planned",
        priority: node.getAttribute("data-pwa-task-priority") || ""
      };
    });
    if (tasks.length > 0) {
      return tasks;
    }
    return [
      { id: "review-launch-checklist", title: "Review launch checklist", time: "09:30", status: "In progress", priority: "High" },
      { id: "send-weekly-update", title: "Send weekly update", time: "11:00", status: "Planned", priority: "Medium" },
      { id: "prepare-payment-mock", title: "Prepare payment mock", time: "15:30", status: "Premium", priority: "Low" }
    ];
  }

  function renderTasks() {
    const list = document.querySelector("[data-pwa-task-list]");
    applySubscriptionState();
    if (!list) {
      return;
    }

    const database = readActiveDatabase();
    const isActive = isSubscriptionActive();
    list.textContent = "";

    database.tasks.forEach(function (task) {
      list.appendChild(createTaskNode(task, isActive));
    });
  }

  function createTaskNode(task, isActive) {
    const article = document.createElement("article");
    article.className = "pwa-task";
    article.setAttribute("data-pwa-task", "true");
    article.setAttribute("data-pwa-task-title", task.title);
    article.setAttribute("data-pwa-task-time", task.time);
    article.setAttribute("data-pwa-task-status", task.status);
    article.setAttribute("data-pwa-task-priority", task.priority);

    const content = document.createElement("div");
    content.className = "pwa-task-main";

    const title = document.createElement("strong");
    title.textContent = task.title;

    const meta = document.createElement("small");
    meta.textContent = [task.time, task.priority].filter(Boolean).join(" · ");

    content.appendChild(title);
    content.appendChild(meta);

    const side = document.createElement("div");
    side.className = "pwa-task-side";

    const status = document.createElement("span");
    status.setAttribute("data-pwa-task-status-label", task.title);
    status.textContent = task.status;
    side.appendChild(status);

    if (isActive) {
      const actions = document.createElement("div");
      actions.className = "pwa-task-actions";
      actions.appendChild(createTaskAction("edit", task.id, readLabel("[data-pwa-edit-label]", "Edit")));
      actions.appendChild(createTaskAction("delete", task.id, readLabel("[data-pwa-delete-label]", "Delete")));
      side.appendChild(actions);
    }

    article.appendChild(content);
    article.appendChild(side);
    return article;
  }

  function createTaskAction(action, id, label) {
    const button = document.createElement("button");
    button.className = "pwa-task-action";
    button.type = "button";
    button.setAttribute("data-pwa-task-action", action);
    button.setAttribute("data-pwa-task-action-id", id);
    button.textContent = label;
    return button;
  }

  function loadTaskIntoForm(id) {
    const form = document.querySelector("[data-pwa-task-form]");
    if (!form) {
      return;
    }

    const database = readActiveDatabase();
    const task = database.tasks.find(function (item) { return item.id === id; });
    if (!task) {
      return;
    }

    form.elements.id.value = task.id;
    form.elements.title.value = task.title;
    form.elements.time.value = task.time;
    form.elements.status.value = task.status;
    form.elements.priority.value = task.priority;
    setTaskMessage("Editing " + task.title + ".");
    form.elements.title.focus();
  }

  function resetTaskForm(form) {
    form.elements.id.value = "";
    setTaskMessage(defaultTaskMessage);
  }

  function applySubscriptionState() {
    const isActive = isSubscriptionActive();
    document.body.classList.toggle("pwa-home-subscribed", isActive && Boolean(document.querySelector("[data-pwa-task-list]")));

    document.querySelectorAll("[data-pwa-requires-subscription]").forEach(function (node) {
      node.hidden = !isActive;
    });

    document.querySelectorAll("[data-pwa-subscription-status]").forEach(function (node) {
      const activeLabel = node.getAttribute("data-pwa-active-label") || node.textContent;
      const inactiveLabel = node.getAttribute("data-pwa-inactive-label") || "Premium inactive";
      node.textContent = isActive ? activeLabel : inactiveLabel;
    });

    const tools = document.querySelector("[data-pwa-task-tools]");
    const form = document.querySelector("[data-pwa-task-form]");
    const lock = document.querySelector("[data-pwa-task-lock]");
    if (!tools || !form) {
      return;
    }

    tools.classList.toggle("pwa-task-tools-locked", !isActive);
    if (lock) {
      lock.hidden = isActive;
    }

    Array.from(form.elements).forEach(function (field) {
      field.disabled = !isActive;
    });
  }

  function activateSubscription() {
    localStorage.setItem(SUBSCRIPTION_KEY, "1");
  }

  function deactivateSubscription() {
    localStorage.setItem(SUBSCRIPTION_KEY, "0");
  }

  function isSubscriptionActive() {
    return localStorage.getItem(SUBSCRIPTION_KEY) === "1";
  }

  async function clearPwaData() {
    if (navigator.serviceWorker && navigator.serviceWorker.controller) {
      navigator.serviceWorker.controller.postMessage({ type: "CLEAR_PWA_CACHE" });
    }

    if ("caches" in window) {
      const keys = await caches.keys();
      await Promise.all(
        keys
          .filter(function (key) { return key.startsWith(CACHE_PREFIX); })
          .map(function (key) { return caches.delete(key); })
      );
    }

    Object.keys(localStorage)
      .filter(function (key) { return key.startsWith(STORAGE_PREFIX); })
      .forEach(function (key) { localStorage.removeItem(key); });

    Object.keys(sessionStorage)
      .filter(function (key) { return key.startsWith(STORAGE_PREFIX); })
      .forEach(function (key) { sessionStorage.removeItem(key); });

    if (navigator.serviceWorker) {
      const registrations = await navigator.serviceWorker.getRegistrations();
      await Promise.all(
        registrations
          .filter(function (registration) {
            return registration.active && registration.active.scriptURL.endsWith("/sw.js");
          })
          .map(function (registration) { return registration.unregister(); })
      );
    }
  }

  function uniqueTaskID(tasks, title) {
    const base = safeID(title) || "task";
    const used = tasks.map(function (task) { return task.id; });
    if (!used.includes(base)) {
      return base;
    }
    let index = 2;
    while (used.includes(base + "-" + index)) {
      index += 1;
    }
    return base + "-" + index;
  }

  function setStatus(message) {
    const target = document.querySelector("[data-pwa-data-status]");
    if (target) {
      target.textContent = message;
    }
  }

  function setTaskMessage(message) {
    const target = document.querySelector("[data-pwa-task-status-message]");
    if (target) {
      target.textContent = message;
    }
  }

  function readLabel(selector, fallback) {
    const node = document.querySelector(selector);
    return node ? node.textContent : fallback;
  }

  function readLockMessage() {
    const node = document.querySelector("[data-pwa-task-lock]");
    return node ? node.textContent : "Activate Premium to edit tasks.";
  }

  function readSubscriptionHint() {
    const node = document.querySelector("[data-pwa-task-status-message]");
    return node ? node.textContent : "";
  }

  function databaseKey(id) {
    return STORAGE_PREFIX + "db:" + safeID(id || DEFAULT_DB_ID);
  }

  function safeID(value) {
    return String(value || "")
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9_-]+/g, "-")
      .replace(/^-+|-+$/g, "")
      .slice(0, 64);
  }
})();
