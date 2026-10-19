import "./style.css";
import * as api from "./api.js";

const CATEGORIES = ["еда", "транспорт", "жильё", "развлечения", "здоровье", "покупки", "другое"];
const PAGE_SIZE_OPTIONS = [10, 20, 50, 100];
const DEFAULT_PAGE_SIZE = 20;

const state = {
  items: [],
  users: [],
  editing: null,
  page: 1,
  pageSize: DEFAULT_PAGE_SIZE,
  totalPages: 0,
  total: 0,
};

const app = document.querySelector("#app");

app.innerHTML = `
  <div class="wrap">
    <h1>Учёт семейных расходов</h1>
    <p class="sub">API: <code id="api-url"></code> · авторизация не нужна</p>

    <section class="panel">
      <div class="filters">
        <label>Пользователь
          <select id="f-user">
            <option value="">все</option>
          </select>
        </label>
        <label>Категория
          <select id="f-category">
            <option value="">все</option>
            ${CATEGORIES.map((c) => `<option value="${c}">${c}</option>`).join("")}
          </select>
        </label>
        <label>С
          <input id="f-from" type="date" />
        </label>
        <label>По
          <input id="f-to" type="date" />
        </label>
        <label>На странице
          <select id="f-page-size">
            ${PAGE_SIZE_OPTIONS.map(
              (n) =>
                `<option value="${n}"${n === DEFAULT_PAGE_SIZE ? " selected" : ""}>${n}</option>`
            ).join("")}
          </select>
        </label>
        <div class="actions">
          <button id="btn-load" type="button">Показать</button>
          <button id="btn-reset" class="secondary" type="button">Сбросить</button>
        </div>
      </div>
    </section>

    <section class="panel">
      <h2 style="margin:0 0 12px;font-size:1.05rem">Новая трата</h2>
      <form id="create-form" class="form-grid">
        <label>Пользователь
          <select name="user_id" id="create-user" required>
            <option value="" disabled selected>выберите</option>
          </select>
        </label>
        <label>Сумма ₽
          <input name="amount" type="number" min="0.01" step="0.01" required />
        </label>
        <label>Категория
          <select name="category" required>
            ${CATEGORIES.map((c) => `<option value="${c}">${c}</option>`).join("")}
          </select>
        </label>
        <label>Дата
          <input name="spent_at" type="date" required />
        </label>
        <label>Комментарий
          <input name="comment" type="text" placeholder="кофе" />
        </label>
        <div class="actions">
          <button type="submit">Добавить</button>
        </div>
      </form>
    </section>

    <p class="status" id="status"></p>
    <p class="total" id="total">Итого: —</p>

    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Дата</th>
            <th>User</th>
            <th>Сумма</th>
            <th>Категория</th>
            <th>Комментарий</th>
            <th></th>
          </tr>
        </thead>
        <tbody id="tbody"></tbody>
      </table>
    </div>

    <div class="pager" id="pager">
      <button type="button" class="secondary" id="btn-prev">Назад</button>
      <span id="pager-info">стр. 1</span>
      <button type="button" class="secondary" id="btn-next">Вперёд</button>
    </div>
  </div>

  <div id="edit-dialog" class="dialog-backdrop hidden">
    <div class="dialog">
      <h2>Редактировать</h2>
      <form id="edit-form" class="form-grid">
        <label>Сумма ₽
          <input name="amount" type="number" min="0.01" step="0.01" required />
        </label>
        <label>Категория
          <select name="category" required>
            ${CATEGORIES.map((c) => `<option value="${c}">${c}</option>`).join("")}
          </select>
        </label>
        <label>Дата
          <input name="spent_at" type="date" required />
        </label>
        <label>Комментарий
          <input name="comment" type="text" />
        </label>
        <div class="actions">
          <button type="submit">Сохранить</button>
          <button type="button" class="secondary" id="btn-cancel-edit">Отмена</button>
        </div>
      </form>
    </div>
  </div>
`;

const statusEl = document.querySelector("#status");
const totalEl = document.querySelector("#total");
const tbody = document.querySelector("#tbody");
const dialog = document.querySelector("#edit-dialog");
const createForm = document.querySelector("#create-form");
const editForm = document.querySelector("#edit-form");
const filterUser = document.querySelector("#f-user");
const pageSizeSelect = document.querySelector("#f-page-size");
const createUser = document.querySelector("#create-user");
const pagerInfo = document.querySelector("#pager-info");
const btnPrev = document.querySelector("#btn-prev");
const btnNext = document.querySelector("#btn-next");

document.querySelector("#api-url").textContent = api.API_URL;
createForm.spent_at.value = todayMoscow();

function todayMoscow() {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: "Europe/Moscow",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date());
}

/** API хранит сумму в копейках. */
function rublesToKopecks(rubles) {
  return Math.round(Number(rubles) * 100);
}

function formatRubles(kopecks) {
  return `${(Number(kopecks) / 100).toFixed(2)} ₽`;
}

function setStatus(msg, isError = false) {
  statusEl.textContent = msg || "";
  statusEl.classList.toggle("error", isError);
}

function currentPageSize() {
  const n = Number(pageSizeSelect.value);
  return PAGE_SIZE_OPTIONS.includes(n) ? n : DEFAULT_PAGE_SIZE;
}

function filters(page = state.page) {
  return {
    user_id: filterUser.value,
    category: document.querySelector("#f-category").value,
    from: document.querySelector("#f-from").value,
    to: document.querySelector("#f-to").value,
    page,
    page_size: currentPageSize(),
  };
}

function fillUserSelects() {
  const selectedFilter = filterUser.value;
  const selectedCreate = createUser.value;

  filterUser.replaceChildren();
  const allOpt = document.createElement("option");
  allOpt.value = "";
  allOpt.textContent = "все";
  filterUser.append(allOpt);

  createUser.replaceChildren();
  const placeholder = document.createElement("option");
  placeholder.value = "";
  placeholder.textContent = "выберите";
  placeholder.disabled = true;
  createUser.append(placeholder);

  for (const id of state.users) {
    const label = String(id);

    const fOpt = document.createElement("option");
    fOpt.value = label;
    fOpt.textContent = label;
    filterUser.append(fOpt);

    const cOpt = document.createElement("option");
    cOpt.value = label;
    cOpt.textContent = label;
    createUser.append(cOpt);
  }

  if (selectedFilter && state.users.map(String).includes(selectedFilter)) {
    filterUser.value = selectedFilter;
  }
  if (selectedCreate && state.users.map(String).includes(selectedCreate)) {
    createUser.value = selectedCreate;
  } else if (state.users.length === 1) {
    createUser.value = String(state.users[0]);
  } else {
    createUser.value = "";
  }
}

function render() {
  tbody.replaceChildren();

  if (!state.items.length) {
    const tr = document.createElement("tr");
    const td = document.createElement("td");
    td.colSpan = 6;
    td.textContent = "Нет записей";
    tr.append(td);
    tbody.append(tr);
    updatePager();
    return;
  }

  for (const e of state.items) {
    const tr = document.createElement("tr");
    tr.dataset.id = e.id;

    const cells = [
      e.spent_at,
      String(e.user_id),
      formatRubles(e.amount),
      e.category,
      e.comment ?? "",
    ];
    for (const text of cells) {
      const td = document.createElement("td");
      td.textContent = text;
      tr.append(td);
    }

    const actions = document.createElement("td");
    actions.className = "row-actions";

    const editBtn = document.createElement("button");
    editBtn.type = "button";
    editBtn.className = "secondary btn-edit";
    editBtn.textContent = "Изменить";

    const delBtn = document.createElement("button");
    delBtn.type = "button";
    delBtn.className = "danger btn-del";
    delBtn.textContent = "Удалить";

    actions.append(editBtn, delBtn);
    tr.append(actions);
    tbody.append(tr);
  }

  updatePager();
}

function updatePager() {
  const totalPages = state.totalPages || 0;
  const page = state.page || 1;
  pagerInfo.textContent =
    totalPages === 0
      ? "нет страниц"
      : `стр. ${page} из ${totalPages} · всего ${state.total}`;
  btnPrev.disabled = page <= 1 || totalPages === 0;
  btnNext.disabled = totalPages === 0 || page >= totalPages;
}

async function loadUsers() {
  const data = await api.listUsers();
  state.users = data.user_ids || [];
  fillUserSelects();
}

async function load(page = state.page) {
  setStatus("Загрузка…");
  totalEl.textContent = "Итого: —";
  try {
    state.page = page;
    const f = filters(page);
    const [data, sumData] = await Promise.all([
      api.listExpenses(f),
      api.sumExpenses(f),
    ]);
    state.items = data.items || [];
    state.page = data.page || page;
    state.pageSize = data.page_size || currentPageSize();
    state.total = Number(data.total ?? 0);
    state.totalPages = Number(data.total_pages ?? 0);
    render();
    totalEl.textContent = `Итого по фильтрам: ${formatRubles(sumData.sum ?? 0)}`;
    setStatus(`Показано: ${state.items.length}`);
  } catch (err) {
    setStatus(err.message, true);
  }
}

document.querySelector("#btn-load").addEventListener("click", () => load(1));
document.querySelector("#btn-reset").addEventListener("click", () => {
  filterUser.value = "";
  document.querySelector("#f-category").value = "";
  document.querySelector("#f-from").value = "";
  document.querySelector("#f-to").value = "";
  pageSizeSelect.value = String(DEFAULT_PAGE_SIZE);
  load(1);
});

pageSizeSelect.addEventListener("change", () => load(1));

btnPrev.addEventListener("click", () => {
  if (state.page > 1) load(state.page - 1);
});
btnNext.addEventListener("click", () => {
  if (state.page < state.totalPages) load(state.page + 1);
});

createForm.addEventListener("submit", async (ev) => {
  ev.preventDefault();
  const fd = new FormData(createForm);
  const body = {
    user_id: Number(fd.get("user_id")),
    amount: rublesToKopecks(fd.get("amount")),
    category: String(fd.get("category")),
    spent_at: String(fd.get("spent_at")),
    comment: String(fd.get("comment") || "").trim() || null,
  };
  if (body.amount <= 0) {
    setStatus("Сумма должна быть больше 0", true);
    return;
  }
  try {
    await api.createExpense(body);
    createForm.reset();
    createForm.spent_at.value = todayMoscow();
    fillUserSelects();
    setStatus("Трата добавлена");
    await loadUsers();
    await load(1);
  } catch (err) {
    setStatus(err.message, true);
  }
});

tbody.addEventListener("click", async (ev) => {
  const btn = ev.target.closest("button");
  if (!btn) return;
  const tr = btn.closest("tr");
  const id = tr?.dataset.id;
  const item = state.items.find((x) => x.id === id);
  if (!item) return;

  if (btn.classList.contains("btn-del")) {
    if (!confirm("Удалить трату?")) return;
    try {
      await api.deleteExpense(id);
      setStatus("Удалено");
      await loadUsers();
      const nextPage =
        state.items.length === 1 && state.page > 1 ? state.page - 1 : state.page;
      await load(nextPage);
    } catch (err) {
      setStatus(err.message, true);
    }
    return;
  }

  if (btn.classList.contains("btn-edit")) {
    state.editing = item;
    editForm.amount.value = (Number(item.amount) / 100).toFixed(2);
    editForm.category.value = item.category;
    editForm.spent_at.value = item.spent_at;
    editForm.comment.value = item.comment ?? "";
    dialog.classList.remove("hidden");
  }
});

document.querySelector("#btn-cancel-edit").addEventListener("click", () => {
  state.editing = null;
  dialog.classList.add("hidden");
});

editForm.addEventListener("submit", async (ev) => {
  ev.preventDefault();
  if (!state.editing) return;
  const fd = new FormData(editForm);
  const body = {
    amount: rublesToKopecks(fd.get("amount")),
    category: String(fd.get("category")),
    spent_at: String(fd.get("spent_at")),
    comment: String(fd.get("comment") || "").trim() || null,
  };
  if (body.amount <= 0) {
    setStatus("Сумма должна быть больше 0", true);
    return;
  }
  try {
    await api.updateExpense(state.editing.id, body);
    state.editing = null;
    dialog.classList.add("hidden");
    setStatus("Сохранено");
    await load(state.page);
  } catch (err) {
    setStatus(err.message, true);
  }
});

(async () => {
  try {
    await loadUsers();
    await load(1);
  } catch (err) {
    setStatus(err.message, true);
  }
})();
