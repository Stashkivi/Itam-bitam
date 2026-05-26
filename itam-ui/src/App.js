import { Fragment as _Fragment, jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { getToken, setToken } from '@/api/client';
import { AppShell } from '@/components/layout/AppShell';
const queryClient = new QueryClient({
    defaultOptions: {
        queries: {
            retry: 1,
            staleTime: 30_000,
        },
    },
});
// ── Simple token gate — keeps auth logic out of AppShell ─────────────────────
function TokenGate({ children }) {
    const [token, setLocalToken] = useState(() => getToken());
    const [input, setInput] = useState('');
    if (token)
        return _jsx(_Fragment, { children: children });
    const submit = (e) => {
        e.preventDefault();
        if (!input.trim())
            return;
        setToken(input.trim());
        setLocalToken(input.trim());
    };
    return (_jsx("div", { className: "flex h-screen items-center justify-center bg-surface", children: _jsxs("form", { onSubmit: submit, className: "w-96 rounded-xl border border-border bg-card p-8 shadow-2xl", children: [_jsx("h1", { className: "text-xl font-bold text-slate-100 mb-1", children: "ITAM Dashboard" }), _jsx("p", { className: "text-sm text-muted mb-6", children: "Enter your access token to continue" }), _jsx("label", { className: "block text-xs text-muted mb-1.5", children: "Access Token" }), _jsx("input", { type: "password", value: input, onChange: (e) => setInput(e.target.value), placeholder: "eyJhbGciOiJIUzI1NiJ9\u2026", className: "\n            w-full rounded-md border border-border bg-surface px-3 py-2\n            text-sm text-slate-200 font-mono placeholder:text-slate-600\n            focus:outline-none focus:ring-2 focus:ring-blue-500\n          " }), _jsx("button", { type: "submit", className: "\n            mt-4 w-full rounded-md bg-blue-600 px-4 py-2 text-sm font-medium\n            text-white hover:bg-blue-500 active:bg-blue-700 transition-colors\n          ", children: "Connect" })] }) }));
}
export default function App() {
    return (_jsx(QueryClientProvider, { client: queryClient, children: _jsx(TokenGate, { children: _jsx(AppShell, {}) }) }));
}
