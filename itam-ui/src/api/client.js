const TOKEN_KEY = 'itam_access_token';
export function getToken() {
    return localStorage.getItem(TOKEN_KEY);
}
export function setToken(token) {
    localStorage.setItem(TOKEN_KEY, token);
}
export function clearToken() {
    localStorage.removeItem(TOKEN_KEY);
}
class ApiError extends Error {
    status;
    constructor(status, message) {
        super(message);
        this.status = status;
        this.name = 'ApiError';
    }
}
async function request(path, init) {
    const token = getToken();
    const headers = {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...init?.headers,
    };
    const res = await fetch(path, { ...init, headers });
    if (res.status === 401) {
        clearToken();
        window.location.reload();
    }
    if (!res.ok) {
        const text = await res.text();
        throw new ApiError(res.status, text);
    }
    if (res.status === 204)
        return undefined;
    return res.json();
}
export const api = {
    get: (path) => request(path),
    post: (path, body) => request(path, { method: 'POST', body: JSON.stringify(body) }),
};
