const API_URL = (import.meta.env.VITE_API_URL || "http://localhost:8080").replace(/\/$/, "");

async function request(path, options = {}) {
  const res = await fetch(`${API_URL}${path}`, {
    headers: {
      "Content-Type": "application/json",
      ...(options.headers || {}),
    },
    ...options,
  });

  if (res.status === 204) {
    return null;
  }

  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error || `HTTP ${res.status}`);
  }
  return data;
}

function toQuery(params = {}) {
  const q = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v !== "" && v != null) q.set(k, v);
  });
  const qs = q.toString();
  return qs ? `?${qs}` : "";
}

export function listExpenses(params = {}) {
  return request(`/api/expenses${toQuery(params)}`);
}

export function sumExpenses(params = {}) {
  const { page, page_size, ...rest } = params;
  return request(`/api/expenses/sum${toQuery(rest)}`);
}

export function listUsers() {
  return request("/api/users");
}

export function createExpense(body) {
  return request("/api/expenses", { method: "POST", body: JSON.stringify(body) });
}

export function updateExpense(id, body) {
  return request(`/api/expenses/${id}`, { method: "PUT", body: JSON.stringify(body) });
}

export function deleteExpense(id) {
  return request(`/api/expenses/${id}`, { method: "DELETE" });
}

export { API_URL };
