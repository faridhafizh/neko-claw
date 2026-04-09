const API_BASE = "http://localhost:8080/api";

export interface Settings {
  apiKey: string;
  model: string;
  apiUrl: string;
}

export interface PendingCommand {
  id: string;
  command: string;
  arguments: string[];
  description: string;
}

export interface ChatResponse {
  reply: string;
  hasPendingCmd: boolean;
  pendingCommand?: PendingCommand;
}

export async function fetchSettings(): Promise<Settings> {
  const res = await fetch(`${API_BASE}/settings`);
  if (!res.ok) throw new Error("Failed to fetch settings");
  return res.json();
}

export async function saveSettings(settings: Settings): Promise<void> {
  const res = await fetch(`${API_BASE}/settings`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(settings)
  });
  if (!res.ok) throw new Error("Failed to save settings");
}

export async function sendChatMessage(message: string): Promise<ChatResponse> {
  const res = await fetch(`${API_BASE}/chat`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message })
  });
  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(errorText || "Failed to send chat message");
  }
  return res.json();
}

export async function approveCommand(id: string): Promise<{status: string, output: string, reply: string}> {
  const res = await fetch(`${API_BASE}/command/approve`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id })
  });
  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(errorText || "Failed to approve command");
  }
  return res.json();
}

export async function rejectCommand(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/command/reject`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id })
  });
  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(errorText || "Failed to reject command");
  }
}
