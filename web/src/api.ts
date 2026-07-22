export interface Account {
  user_id: string;
  nickname: string;
  remark: string;
  api_token: string;
  is_default: boolean;
  default_model: string;
  created_at?: string;
  display_order: number;
  active_sessions: number;
  total_requests: number;
  today_requests: number;
  total_tokens: number;
  today_tokens: number;
  credential_valid: number; // -1=unknown, 0=expired, 1=valid
  credential_checked_at?: string;
  credential_refreshed_at?: string;
  credential_error?: string;
}

export function accountDisplayName(a: { nickname?: string; remark?: string; user_id: string }): string {
  if (a.remark) return a.remark;
  if (a.nickname) return a.nickname;
  return a.user_id;
}

export interface ModelInfo {
  id: string;
  name: string;
}

export interface ModelCatalogItem {
  id: string;
  name: string;
  provider: string;
  type: string; // chat, reasoning, vision
  context_window: number;
  free: boolean;
  default: boolean;
  effort_applicable: boolean;
  input_price: string;
  output_price: string;
  pricing_note?: string;
  max_output_tokens?: number;
}

export interface Stats {
  total_requests: number;
  total_input_tokens: number;
  total_output_tokens: number;
  accounts_count: number;
  avg_latency_ms: number;
  error_count: number;
  stream_count: number;
  success_count: number;
  by_model: { model: string; count: number }[];
  by_account: { user_id: string; nickname: string; remark: string; count: number }[];
  all_time: {
    total_requests: number;
    total_input_tokens: number;
    total_output_tokens: number;
    error_count: number;
  };
  hourly: {
    hour: string;
    count: number;
    input_tokens: number;
    output_tokens: number;
    errors: number;
  }[];
}

export interface Settings {
  [key: string]: string;
}

export interface AccountStats {
  user_id: string;
  nickname: string;
  remark: string;
  total_requests: number;
  total_input_tokens: number;
  total_output_tokens: number;
  success_count: number;
  stream_count: number;
  by_model: { model: string; count: number }[];
  by_endpoint: { endpoint: string; count: number }[];
  avg_latency_ms: number;
  error_count: number;
  all_time?: {
    total_requests: number;
    total_input_tokens: number;
    total_output_tokens: number;
    error_count: number;
  };
  hourly: {
    hour: string;
    count: number;
    input_tokens: number;
    output_tokens: number;
    errors: number;
  }[];
}

export interface RequestLog {
  id: number;
  user_id: string;
  model: string;
  endpoint: string;
  stream: boolean;
  status_code: number;
  latency_ms: number;
  error_message: string;
  input_tokens: number;
  output_tokens: number;
  created_at: string;
}

const TOKEN_KEY = 'atomcode_jwt';

function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY);
}

export function isAuthenticated(): boolean {
  return !!getToken();
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  const token = getToken();
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  const resp = await fetch(path, {
    headers,
    ...options,
  });
  if (resp.status === 401) {
    clearToken();
    window.location.href = '/login';
    throw new Error('Unauthorized');
  }
  if (!resp.ok) {
    const err = await resp.json().catch(() => ({ detail: resp.statusText }));
    throw new Error(err.detail || `HTTP ${resp.status}`);
  }
  return resp.json();
}

async function authRequest<T>(path: string, options?: RequestInit): Promise<T> {
  const resp = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!resp.ok) {
    const err = await resp.json().catch(() => ({ detail: resp.statusText }));
    throw new Error(err.detail || `HTTP ${resp.status}`);
  }
  return resp.json();
}

export const authApi = {
  status: () => authRequest<{ initialized: boolean; exe_path?: string }>('/api/auth/status'),
  setup: (password: string) =>
    authRequest<{ ok: boolean; token: string }>('/api/auth/setup', {
      method: 'POST',
      body: JSON.stringify({ password }),
    }),
  login: (password: string) =>
    authRequest<{ ok: boolean; token: string }>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ password }),
    }),
  changePassword: (oldPassword: string, newPassword: string) =>
    request<{ ok: boolean }>('/api/auth/change-password', {
      method: 'POST',
      body: JSON.stringify({ old_password: oldPassword, new_password: newPassword }),
    }),
};

export const api = {
  listAccounts: () => request<{ accounts: Account[] }>('/api/accounts').then(r => r.accounts),
  addAccount: (data: { user_id: string; pt_key: string; nickname?: string; is_default?: boolean; default_model?: string }) =>
    request<{ ok: boolean }>('/api/accounts', { method: 'POST', body: JSON.stringify(data) }),
  removeAccount: (userId: string) =>
    request<{ ok: boolean }>(`/api/accounts/${encodeURIComponent(userId)}`, { method: 'DELETE' }),
  setDefault: (userId: string) =>
    request<{ ok: boolean }>(`/api/accounts/${encodeURIComponent(userId)}/default`, { method: 'PUT' }),
  validateAccount: (userId: string) =>
    request<{ valid: boolean }>(`/api/accounts/${encodeURIComponent(userId)}/validate`, { method: 'POST' }),
  listModels: () => request<{ models: ModelInfo[] }>('/api/models').then(r => r.models),
  listAccountModels: (userId: string) =>
    request<{ models: ModelInfo[] }>(`/api/accounts/${encodeURIComponent(userId)}/models`).then(r => r.models),
  getStats: () => request<Stats>('/api/stats'),
  getSettings: () => request<{ settings: Settings }>('/api/settings').then(r => r.settings),
  updateSettings: (data: Settings) =>
    request<{ ok: boolean }>('/api/settings', { method: 'PUT', body: JSON.stringify(data) }),
  getHealth: () => request<{ status: string; accounts: number }>('/api/health'),
  updateAccountModel: (userId: string, defaultModel: string) =>
    request<{ ok: boolean }>(`/api/accounts/${encodeURIComponent(userId)}/model`, {
      method: 'PUT',
      body: JSON.stringify({ default_model: defaultModel }),
    }),
  getAccountStats: (userId: string) =>
    request<AccountStats>(`/api/accounts/${encodeURIComponent(userId)}/stats`),
  getAccountLogs: (userId: string, limit = 200) =>
    request<{ logs: RequestLog[]; total: number }>(`/api/accounts/${encodeURIComponent(userId)}/logs?limit=${limit}`),
  renewToken: (userId: string) =>
    request<{ ok: boolean; api_token: string }>(`/api/accounts/${encodeURIComponent(userId)}/renew-token`, { method: 'POST' }),
  autoLogin: () =>
    request<{ ok: boolean; user_id: string; nickname: string; real_name: string; is_default: boolean }>('/api/accounts-auto-login', { method: 'POST' }),
  qrLoginInit: () =>
    request<{ ok: boolean; session_id: string; qr_image: string }>('/api/qr-login/init', { method: 'POST' }),
  qrLoginStatus: (sessionId: string) =>
    request<{ status: string; ok?: boolean; user_id?: string; nickname?: string; real_name?: string; message?: string; verify_url?: string; risk_code?: number }>(`/api/qr-login/status?session=${encodeURIComponent(sessionId)}`),
  browserLogin: () =>
    request<{ ok: boolean; url: string; login_id: string; expires_in_seconds: number }>('/api/browser-login', { method: 'POST' }),
  oauthPoll: (loginId: string) =>
    request<{ status: string; ok?: boolean; user_id?: string; nickname?: string; message?: string; error?: string }>('/api/oauth-callback', {
      method: 'POST',
      body: JSON.stringify({ login_id: loginId }),
    }),
  oauthSubmit: (ptKey: string) =>
    request<{ ok: boolean; user_id: string; nickname: string }>('/api/oauth-submit', { method: 'POST', body: JSON.stringify({ pt_key: ptKey }) }),
  getRecentErrors: (limit = 50) =>
    request<{ errors: RequestLog[]; total: number }>(`/api/errors?limit=${limit}`),
  clearAllAccounts: () =>
    request<{ ok: boolean; count: number }>('/api/accounts-clear-all', { method: 'POST' }),
  updateRemark: (userId: string, remark: string) =>
    request<{ ok: boolean }>(`/api/accounts/${encodeURIComponent(userId)}/remark`, { method: 'PUT', body: JSON.stringify({ remark }) }),
  exportAccounts: () =>
    request<{ ok: boolean; accounts: Array<{ user_id: string; nickname: string; remark: string; pt_key: string; is_default: boolean; default_model: string; display_order: number }>; count: number }>('/api/accounts-export'),
  importAccounts: (accounts: Array<{ user_id: string; nickname: string; remark: string; pt_key: string; is_default: boolean; default_model: string; display_order: number }>) =>
    request<{ ok: boolean; added: number; updated: number; total: number }>('/api/accounts-import', { method: 'POST', body: JSON.stringify({ accounts }) }),
  listModelsCatalog: () =>
    request<{ catalog: ModelCatalogItem[] }>('/api/models/catalog').then(r => r.catalog),
  getCodingPlanStatus: () =>
    request<{
      plan: { name: string; expires_at: string; remaining_days: number; total_days: number };
      usage: { current_window_percent: number; resets_at: string; reset_label: string };
      free_models: string[];
      paid_models: string[];
      note: string;
    }>('/api/codingplan/status'),
};
