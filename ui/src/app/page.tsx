"use client";

import { useState, useEffect, useRef, useCallback, KeyboardEvent } from "react";
import { Markdown } from "@/components/Markdown";
import {
  streamChatMessage,
  sendChatMessage,
  approveCommand,
  rejectCommand,
  PendingCommand,
  ChatSession,
  fetchChatSessions,
  createChatSession,
  fetchChatSession,
  updateChatSession,
  fetchSystemInfo,
  SystemInfo,
} from "@/lib/api";

// ─── Types ─────────────────────────────────────────────────────────────────

type MessageRole = "user" | "ai" | "system";

interface Message {
  role: MessageRole;
  text: string;
  timestamp: Date;
}

// Removed basic renderMarkdown in favor of the rich Markdown component.

// ─── Helpers ────────────────────────────────────────────────────────────────

function formatTime(dateStr: string | Date): string {
  const d = typeof dateStr === "string" ? new Date(dateStr) : dateStr;
  const now = new Date();
  const diffMs = now.getTime() - d.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return "Just now";
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  if (diffHr < 48) return "Yesterday";
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

/*
function formatMsgTime(d: Date): string {
  return d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
}
*/ // Remove formatMsgTime – it’s no longer used

// Use a stable factory so the timestamp is always set at interaction time,
// never at module-load time (which can differ between server and client).
function makeWelcomeMsg(): Message {
  return {
    role: "system",
    text: "Welcome to Neko AI Controller 🐾. Meow can I help you today?",
    timestamp: new Date(),
  };
}

// ─── Component ──────────────────────────────────────────────────────────────

export default function Home() {
  const [messages, setMessages] = useState<Message[]>(() => [makeWelcomeMsg()]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [pendingCmd, setPendingCmd] = useState<PendingCommand | null>(null);

  // Soul / memory info from last API response
  const [activeSoul, setActiveSoul] = useState("Default Neko");
  const [soulEmoji, setSoulEmoji] = useState("🐱");
  const [memoryCount, setMemoryCount] = useState(0);

  // Session state
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [showArchived, setShowArchived] = useState(false);
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [sessionsLoading, setSessionsLoading] = useState(false);
  const [sysInfo, setSysInfo] = useState<SystemInfo | null>(null);

  // Inline rename
  const [editingSessionId, setEditingSessionId] = useState<string | null>(null);
  const [editingTitle, setEditingTitle] = useState("");

  // Sidebar search
  const [searchQuery, setSearchQuery] = useState("");

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  // ── Load sessions ──────────────────────────────────────────────────────────

  const loadSessions = useCallback(async () => {
    setSessionsLoading(true);
    try {
      const data = await fetchChatSessions(showArchived);
      setSessions(data.sessions);
    } catch (e) {
      console.error("Failed to load sessions:", e);
    } finally {
      setSessionsLoading(false);
    }
  }, [showArchived]);

  useEffect(() => {
    loadSessions();
  }, [loadSessions]);

  // ── Auto-scroll to bottom ──────────────────────────────────────────────────

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, loading]);

  // ── Keyboard shortcuts ─────────────────────────────────────────────────────

  useEffect(() => {
    const handler = (e: globalThis.KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "n") {
        e.preventDefault();
        newChat();
      }
    };
    window.addEventListener("keydown", handler);

    // Fetch system info periodically
    const fetchSys = async () => {
      try {
        const info = await fetchSystemInfo();
        setSysInfo(info);
      } catch {
        // Ignore silent fails for background polling
      }
    };
    fetchSys();
    const sysInterval = setInterval(fetchSys, 10000);

    return () => {
      window.removeEventListener("keydown", handler);
      clearInterval(sysInterval);
    };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // ── Session actions ────────────────────────────────────────────────────────

  const selectSession = async (id: string) => {
    if (id === sessionId) return;
    try {
      const session = await fetchChatSession(id);
      setSessionId(id);
      const uiMessages: Message[] = session.messages
        .filter((m) => ["user", "assistant", "system"].includes(m.role))
        .map((m) => ({
          role: m.role === "assistant" ? "ai" : (m.role as MessageRole),
          text: m.content,
          timestamp: m.createdAt
            ? new Date(m.createdAt)
            : new Date(session.updatedAt),
        }));
      setMessages(uiMessages.length > 0 ? uiMessages : [makeWelcomeMsg()]);
      setPendingCmd(null);
    } catch (e) {
      console.error("Failed to load session:", e);
    }
  };

  const newChat = async () => {
    try {
      const session = await createChatSession();
      setSessionId(session.id);
      setMessages([makeWelcomeMsg()]);
      setPendingCmd(null);
      setInput("");
      await loadSessions();
      inputRef.current?.focus();
    } catch (e) {
      console.error("Failed to create session:", e);
    }
  };

  const handleArchiveSession = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      const session = sessions.find((s) => s.id === id);
      await updateChatSession(id, session?.archived ? "unarchive" : "archive");
      if (sessionId === id) setSessionId(null);
      await loadSessions();
    } catch (e) {
      console.error(e);
    }
  };

  const handleDeleteSession = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    if (!confirm("Delete this chat? This cannot be undone.")) return;
    try {
      await updateChatSession(id, "delete");
      if (sessionId === id) {
        setSessionId(null);
        setMessages([makeWelcomeMsg()]);
      }
      await loadSessions();
    } catch (e) {
      console.error(e);
    }
  };

  const handleRenameSession = async (id: string) => {
    if (!editingTitle.trim()) return;
    try {
      await updateChatSession(id, "rename", editingTitle);
      setEditingSessionId(null);
      setEditingTitle("");
      await loadSessions();
    } catch (e) {
      console.error(e);
    }
  };

  // ── Send message ───────────────────────────────────────────────────────────

  const sendMessage = async () => {
    if (!input.trim() || loading || pendingCmd) return;
    const msg = input.trim();
    setInput("");
    setMessages((prev) => [
      ...prev,
      { role: "user", text: msg, timestamp: new Date() },
    ]);
    setLoading(true);

    // Add an empty AI message that will be filled by streaming tokens
    const aiMsgIndex = { current: -1 };
    setMessages((prev) => {
      aiMsgIndex.current = prev.length; // index of the new AI message
      return [
        ...prev,
        { role: "ai" as MessageRole, text: "", timestamp: new Date() },
      ];
    });

    try {
      await streamChatMessage(
        msg,
        {
          onToken: (token: string) => {
            setMessages((prev) => {
              const updated = [...prev];
              const idx = aiMsgIndex.current;
              if (idx >= 0 && idx < updated.length) {
                updated[idx] = {
                  ...updated[idx],
                  text: updated[idx].text + token,
                };
              }
              return updated;
            });
          },
          onToolCall: (cmd: PendingCommand) => {
            setPendingCmd(cmd);
          },
          onDone: (meta) => {
            setActiveSoul(meta.activeSoul);
            setSoulEmoji(meta.soulEmoji);
            setMemoryCount(meta.memoryCount);

            if (!sessionId && meta.sessionId) {
              setSessionId(meta.sessionId);
              loadSessions();
            } else {
              setSessions((prev) =>
                prev.map((s) =>
                  s.id === (sessionId || meta.sessionId)
                    ? {
                        ...s,
                        updatedAt: new Date().toISOString(),
                        lastMessage: msg,
                      }
                    : s,
                ),
              );
            }
          },
          onError: (errMsg: string) => {
            setMessages((prev) => [
              ...prev,
              {
                role: "system",
                text: "Error: " + errMsg,
                timestamp: new Date(),
              },
            ]);
          },
        },
        sessionId || undefined,
      );
    } catch {
      // Fallback: try non-streaming
      try {
        const res = await sendChatMessage(msg, sessionId || undefined);
        setMessages((prev) => {
          const updated = [...prev];
          const idx = aiMsgIndex.current;
          if (idx >= 0 && idx < updated.length) {
            updated[idx] = { ...updated[idx], text: res.reply };
          }
          return updated;
        });
        setActiveSoul(res.activeSoul);
        setSoulEmoji(res.soulEmoji);
        setMemoryCount(res.memoryCount);
        if (!sessionId && res.sessionId) {
          setSessionId(res.sessionId);
          await loadSessions();
        }
        if (res.hasPendingCmd && res.pendingCommand) {
          setPendingCmd(res.pendingCommand);
        }
      } catch (fallbackErr: unknown) {
        const errText =
          fallbackErr instanceof Error ? fallbackErr.message : "Unknown error";
        setMessages((prev) => [
          ...prev,
          { role: "system", text: "Error: " + errText, timestamp: new Date() },
        ]);
      }
    }
    setLoading(false);
  };

  const handleInputKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  // ── Approve / Reject ───────────────────────────────────────────────────────

  const handleApprove = async () => {
    if (!pendingCmd) return;
    setLoading(true);
    const cmdId = pendingCmd.id;
    setPendingCmd(null);
    try {
      const res = await approveCommand(cmdId);
      setMessages((prev) => [
        ...prev,
        {
          role: "system",
          text: `✅ Command executed:\n${res.output}`,
          timestamp: new Date(),
        },
        { role: "ai", text: res.reply, timestamp: new Date() },
      ]);
      if (res.sessionId && !sessionId) setSessionId(res.sessionId);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Unknown error";
      setMessages((prev) => [
        ...prev,
        {
          role: "system",
          text: "Error executing: " + msg,
          timestamp: new Date(),
        },
      ]);
    }
    setLoading(false);
  };

  const handleReject = async () => {
    if (!pendingCmd) return;
    setLoading(true);
    const cmdId = pendingCmd.id;
    setPendingCmd(null);
    try {
      await rejectCommand(cmdId);
      setMessages((prev) => [
        ...prev,
        { role: "system", text: "❌ Command rejected.", timestamp: new Date() },
      ]);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Unknown error";
      setMessages((prev) => [
        ...prev,
        { role: "system", text: "Error: " + msg, timestamp: new Date() },
      ]);
    }
    setLoading(false);
  };

  // ── Filtered sessions ──────────────────────────────────────────────────────

  const filteredSessions = sessions.filter(
    (s) =>
      s.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      (s.lastMessage || "").toLowerCase().includes(searchQuery.toLowerCase()),
  );

  // ── Render ─────────────────────────────────────────────────────────────────

  return (
    <div className="flex h-[calc(100vh-57px)] overflow-hidden">
      {/* ── Sidebar ── */}
      <div
        className={`flex flex-col bg-stone-900 border-r border-stone-800 transition-all duration-300 ease-in-out overflow-hidden ${
          sidebarOpen ? "w-72 min-w-[288px]" : "w-0 min-w-0"
        }`}
      >
        {/* Sidebar header */}
        <div className="p-3 border-b border-stone-800 shrink-0">
          <button
            id="new-chat-btn"
            onClick={newChat}
            className="w-full bg-amber-500 hover:bg-amber-400 active:scale-95 text-stone-950 font-bold py-2.5 px-4 rounded-xl transition-all flex items-center justify-center gap-2 shadow-md shadow-amber-900/30 animate-pulse-glow"
          >
            <span className="text-base">✦</span> New Chat
            <span className="ml-auto text-xs font-normal opacity-60">
              Ctrl+N
            </span>
          </button>

          {/* Active / Archived toggle */}
          <div className="flex mt-2.5 gap-1 bg-stone-800 rounded-lg p-1">
            <button
              onClick={() => setShowArchived(false)}
              className={`flex-1 text-xs py-1.5 rounded-md font-semibold transition-all ${
                !showArchived
                  ? "bg-amber-500 text-stone-950 shadow-sm"
                  : "text-stone-400 hover:text-stone-200"
              }`}
            >
              Active
            </button>
            <button
              onClick={() => setShowArchived(true)}
              className={`flex-1 text-xs py-1.5 rounded-md font-semibold transition-all ${
                showArchived
                  ? "bg-amber-500 text-stone-950 shadow-sm"
                  : "text-stone-400 hover:text-stone-200"
              }`}
            >
              Archived
            </button>
          </div>

          {/* Search */}
          <div className="relative mt-2">
            <span className="absolute left-2.5 top-1/2 -translate-y-1/2 text-stone-500 text-xs">
              🔍
            </span>
            <input
              type="text"
              placeholder="Search chats…"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full bg-stone-800 border border-stone-700 focus:border-amber-500 rounded-lg text-xs pl-7 pr-3 py-2 text-stone-300 placeholder-stone-500 focus:outline-none transition-colors"
            />
          </div>
        </div>

        {/* Session list */}
        <div className="flex-1 overflow-y-auto p-2 space-y-1">
          {sessionsLoading ? (
            // Skeleton loader
            Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="p-3 rounded-xl animate-pulse">
                <div className="h-3 bg-stone-800 rounded-full w-3/4 mb-2" />
                <div className="h-2.5 bg-stone-800 rounded-full w-1/2" />
              </div>
            ))
          ) : filteredSessions.length === 0 ? (
            <div className="text-center py-10 px-4">
              <span className="text-3xl block mb-2">
                {searchQuery ? "🔍" : "🐾"}
              </span>
              <p className="text-stone-500 text-xs">
                {searchQuery
                  ? "No matches found"
                  : "No chats yet. Start a new one!"}
              </p>
            </div>
          ) : (
            filteredSessions.map((s) => (
              <div
                key={s.id}
                id={`session-${s.id}`}
                onClick={() => selectSession(s.id)}
                className={`p-3 rounded-xl cursor-pointer transition-all group animate-fade-slide-left ${
                  sessionId === s.id
                    ? "bg-amber-500/15 border border-amber-500/40 shadow-sm"
                    : "hover:bg-stone-800 border border-transparent"
                }`}
              >
                {editingSessionId === s.id ? (
                  <div
                    className="flex gap-1"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <input
                      type="text"
                      value={editingTitle}
                      onChange={(e) => setEditingTitle(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") handleRenameSession(s.id);
                        if (e.key === "Escape") {
                          setEditingSessionId(null);
                          setEditingTitle("");
                        }
                      }}
                      className="flex-1 text-xs bg-stone-900 border border-amber-500/50 rounded-lg px-2 py-1.5 text-stone-200 focus:outline-none"
                      autoFocus
                    />
                    <button
                      onClick={() => handleRenameSession(s.id)}
                      className="text-green-400 hover:text-green-300 text-lg leading-none px-1"
                      title="Save"
                    >
                      ✓
                    </button>
                    <button
                      onClick={() => {
                        setEditingSessionId(null);
                        setEditingTitle("");
                      }}
                      className="text-red-400 hover:text-red-300 text-lg leading-none px-1"
                      title="Cancel"
                    >
                      ✕
                    </button>
                  </div>
                ) : (
                  <>
                    <div className="flex items-start justify-between gap-1">
                      <p
                        className={`text-xs font-semibold truncate flex-1 leading-snug ${
                          sessionId === s.id
                            ? "text-amber-300"
                            : "text-stone-200"
                        }`}
                      >
                        {s.archived && (
                          <span className="mr-1 opacity-60">📦</span>
                        )}
                        {s.title}
                      </p>
                      {/* Action buttons — shown on hover */}
                      <div className="flex gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity shrink-0">
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            setEditingSessionId(s.id);
                            setEditingTitle(s.title);
                          }}
                          className="w-5 h-5 flex items-center justify-center text-stone-500 hover:text-amber-400 rounded text-xs transition-colors"
                          title="Rename"
                        >
                          ✎
                        </button>
                        <button
                          onClick={(e) => handleArchiveSession(s.id, e)}
                          className="w-5 h-5 flex items-center justify-center text-stone-500 hover:text-amber-400 rounded text-xs transition-colors"
                          title={s.archived ? "Unarchive" : "Archive"}
                        >
                          {s.archived ? "📤" : "📥"}
                        </button>
                        <button
                          onClick={(e) => handleDeleteSession(s.id, e)}
                          className="w-5 h-5 flex items-center justify-center text-stone-500 hover:text-red-400 rounded text-xs transition-colors"
                          title="Delete"
                        >
                          🗑
                        </button>
                      </div>
                    </div>

                    {/* Last message preview */}
                    {s.lastMessage && (
                      <p className="text-xs text-stone-500 mt-1 truncate leading-snug">
                        {s.lastMessage}
                      </p>
                    )}

                    {/* Footer: time + message count */}
                    <div className="flex items-center justify-between mt-1.5">
                      <span className="text-[10px] text-stone-600">
                        {formatTime(s.updatedAt)}
                      </span>
                      {s.messageCount != null && s.messageCount > 0 && (
                        <span className="text-[10px] text-stone-600 bg-stone-800 px-1.5 py-0.5 rounded-full">
                          {s.messageCount} msg{s.messageCount !== 1 ? "s" : ""}
                        </span>
                      )}
                    </div>
                  </>
                )}
              </div>
            ))
          )}
        </div>
      </div>

      {/* ── Main Chat Area ── */}
      <div className="flex-1 flex flex-col bg-stone-950 overflow-hidden">
        {/* Status bar */}
        <div className="bg-stone-900/60 backdrop-blur border-b border-stone-800 px-4 py-2 flex items-center justify-between shrink-0">
          <div className="flex items-center gap-3">
            <button
              id="sidebar-toggle"
              onClick={() => setSidebarOpen(!sidebarOpen)}
              className="text-stone-500 hover:text-amber-400 transition-colors p-1 rounded-lg hover:bg-stone-800"
              title={sidebarOpen ? "Hide sidebar" : "Show sidebar"}
            >
              {sidebarOpen ? (
                <svg
                  width="16"
                  height="16"
                  viewBox="0 0 16 16"
                  fill="currentColor"
                >
                  <path d="M1 2h14v1.5H1V2zm0 5h9v1.5H1V7zm0 5h14v1.5H1V12z" />
                  <path d="M14 5.5 11 8l3 2.5V5.5z" />
                </svg>
              ) : (
                <svg
                  width="16"
                  height="16"
                  viewBox="0 0 16 16"
                  fill="currentColor"
                >
                  <path d="M1 2h14v1.5H1V2zm0 5h9v1.5H1V7zm0 5h14v1.5H1V12z" />
                  <path d="M9 5.5 12 8l-3 2.5V5.5z" />
                </svg>
              )}
            </button>
            <div className="flex items-center gap-2">
              <span className="text-2xl leading-none">{soulEmoji}</span>
              <div>
                <p className="text-xs font-bold text-stone-200 leading-none">
                  {activeSoul}
                </p>
                <p className="text-[10px] text-stone-500 leading-none mt-0.5">
                  Active Soul
                </p>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-4">
            {/* System Info Widget */}
            {sysInfo && (
              <div className="hidden md:flex items-center gap-3 bg-stone-900 border border-stone-800 rounded-lg px-3 py-1.5 shadow-inner">
                <div className="flex flex-col">
                  <span className="text-[9px] text-stone-500 font-bold uppercase track-wider">
                    {sysInfo.hostname}
                  </span>
                  <span className="text-[10px] text-stone-400 font-mono">
                    {sysInfo.os.split(" ")[0]}
                  </span>
                </div>
                <div className="h-6 w-px bg-stone-800"></div>
                <div className="flex items-center gap-1.5" title="CPU Usage">
                  <div
                    className="w-1.5 h-1.5 rounded-full"
                    style={{
                      backgroundColor:
                        sysInfo.cpuUsage > 80
                          ? "#ef4444"
                          : sysInfo.cpuUsage > 50
                            ? "#f59e0b"
                            : "#10b981",
                    }}
                  ></div>
                  <span className="text-[10px] text-stone-300 font-mono w-6">
                    {sysInfo.cpuUsage}%
                  </span>
                </div>
                <div className="flex items-center gap-1.5" title="Memory Usage">
                  <div
                    className="w-1.5 h-1.5 rounded-full"
                    style={{
                      backgroundColor:
                        sysInfo.ramUsage > 85
                          ? "#ef4444"
                          : sysInfo.ramUsage > 60
                            ? "#f59e0b"
                            : "#10b981",
                    }}
                  ></div>
                  <span className="text-[10px] text-stone-300 font-mono w-7">
                    {sysInfo.ramUsage}%
                  </span>
                </div>
              </div>
            )}

            <div className="flex items-center gap-3 text-xs border-l border-stone-800 pl-4">
              <span className="text-stone-500 flex items-center gap-1">
                🧠{" "}
                <span className="text-stone-300 font-semibold">
                  {memoryCount}
                </span>
                <span className="hidden sm:inline text-stone-600">
                  memories
                </span>
              </span>
              <a
                href="/memory"
                className="px-2.5 py-1 rounded-lg bg-stone-800 hover:bg-amber-500/20 hover:text-amber-400 text-stone-400 font-medium transition-all border border-stone-700 hover:border-amber-500/40"
              >
                Manage
              </a>
            </div>
          </div>
        </div>

        {/* Messages */}
        <div className="flex-1 p-4 overflow-y-auto flex flex-col gap-3">
          {messages.map((m, i) => (
            <MessageBubble
              key={`${m.role}-${i}`}
              message={m}
              soulEmoji={soulEmoji}
            />
          ))}

          {/* Typing indicator */}
          {loading && (
            <div className="flex items-center gap-3 self-start animate-fade-slide-in">
              <span className="text-xl">{soulEmoji}</span>
              <div className="bg-stone-800 border border-stone-700 rounded-2xl rounded-bl-none px-4 py-3">
                <div className="typing-indicator">
                  <span />
                  <span />
                  <span />
                </div>
              </div>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>

        {/* Pending command panel */}
        {pendingCmd && (
          <div className="px-4 py-4 bg-stone-900 border-t-2 border-red-500/60 animate-from-bottom shrink-0">
            <div className="flex items-center gap-2 mb-3">
              <span className="text-lg">🙀</span>
              <p className="text-red-400 font-bold text-sm">
                Neko wants to run a command
              </p>
            </div>
            <div className="bg-stone-950 border border-stone-700 rounded-xl p-3 font-mono text-sm text-emerald-400 mb-2 overflow-x-auto">
              {pendingCmd.command}
            </div>
            <p className="text-stone-400 text-xs italic mb-4 pl-1">
              &quot;{pendingCmd.description}&quot;
            </p>
            <div className="flex gap-3">
              <button
                id="approve-btn"
                onClick={handleApprove}
                className="px-5 py-2.5 bg-emerald-600 hover:bg-emerald-500 active:scale-95 rounded-xl text-white text-sm font-bold transition-all shadow-lg shadow-emerald-900/30"
              >
                ✅ Purr-fect (Approve)
              </button>
              <button
                id="reject-btn"
                onClick={handleReject}
                className="px-5 py-2.5 bg-stone-800 hover:bg-red-900/60 border border-red-500/30 hover:border-red-500/60 active:scale-95 rounded-xl text-red-400 hover:text-red-300 text-sm font-bold transition-all"
              >
                ❌ Hiss (Reject)
              </button>
            </div>
          </div>
        )}

        {/* Input area */}
        <div className="px-4 py-3 bg-stone-900/60 border-t border-stone-800 shrink-0">
          <div className="flex gap-2 items-end">
            <textarea
              ref={inputRef}
              id="chat-input"
              disabled={loading || !!pendingCmd}
              rows={1}
              className="flex-1 bg-stone-800 border border-stone-700 focus:border-amber-500/60 rounded-2xl px-4 py-3 text-sm focus:outline-none transition-colors disabled:opacity-40 text-stone-100 placeholder-stone-500 resize-none min-h-[48px] max-h-32 overflow-y-auto leading-relaxed shadow-inner"
              placeholder="Ask Neko to do something… (Enter to send, Shift+Enter for newline)"
              value={input}
              onChange={(e) => {
                setInput(e.target.value);
                // Auto-resize
                e.target.style.height = "auto";
                e.target.style.height =
                  Math.min(e.target.scrollHeight, 128) + "px";
              }}
              onKeyDown={handleInputKeyDown}
            />
            <button
              id="send-btn"
              disabled={loading || !!pendingCmd || !input.trim()}
              onClick={sendMessage}
              className="bg-amber-500 hover:bg-amber-400 disabled:opacity-30 disabled:cursor-not-allowed active:scale-95 px-5 py-3 rounded-2xl font-bold text-stone-950 transition-all shadow-md shadow-amber-900/30 shrink-0 text-sm"
            >
              Send 🐾
            </button>
          </div>
          <p className="text-[10px] text-stone-700 mt-1.5 pl-1">
            Enter to send · Shift+Enter for new line · Ctrl+N for new chat
          </p>
        </div>
      </div>
    </div>
  );
}

// ─── MessageBubble ─────────────────────────────────────────────────────────

interface MessageBubbleProps {
  message: Message;
  soulEmoji: string;
}

function MessageBubble({ message: m, soulEmoji }: MessageBubbleProps) {
  // Store formatted time in state, initially null (same on server & client)
  const [formattedTime, setFormattedTime] = useState<string | null>(null);

  useEffect(() => {
    // This runs only on the client after hydration
    setFormattedTime(
      m.timestamp.toLocaleTimeString(undefined, {
        hour: "2-digit",
        minute: "2-digit",
      }),
    );
  }, [m.timestamp]);

  // Helper to render the timestamp line (used by all roles)
  const renderTimestamp = () => (
    <p className="text-[10px] text-stone-700 text-center mt-1">
      {formattedTime ?? "⌛"}
    </p>
  );

  if (m.role === "system") {
    return (
      <div className="self-center max-w-[85%] animate-fade-slide-in">
        <div className="bg-stone-800/60 border border-stone-700 rounded-xl px-4 py-2.5 text-xs text-stone-400 font-mono whitespace-pre-wrap text-center">
          {m.text}
        </div>
        {renderTimestamp()}
      </div>
    );
  }

  if (m.role === "user") {
    return (
      <div className="self-end max-w-[80%] animate-fade-slide-right">
        <div className="bg-amber-500 rounded-2xl rounded-br-none px-4 py-3 shadow-md shadow-amber-900/20">
          <p className="text-xs font-bold text-amber-950 mb-1">You</p>
          <p className="text-sm text-stone-950 whitespace-pre-wrap leading-relaxed">
            {m.text}
          </p>
        </div>
        <p className="text-[10px] text-stone-600 text-right mt-1 pr-1">
          {formattedTime ?? "⌛"}
        </p>
      </div>
    );
  }

  // AI message
  return (
    <div className="self-start max-w-[80%] animate-fade-slide-left">
      <div className="flex items-start gap-2.5">
        <span className="text-xl mt-0.5 shrink-0">{soulEmoji}</span>
        <div>
          <div className="bg-stone-800 border border-stone-700 rounded-2xl rounded-tl-none px-4 py-3 shadow-sm">
            <p className="text-xs font-bold text-amber-400 mb-4">Neko AI</p>
            <div className="text-sm text-stone-200 leading-relaxed message-content">
              <Markdown content={m.text} />
            </div>
          </div>
          {renderTimestamp()}
        </div>
      </div>
    </div>
  );
}

/*
interface MessageBubbleProps {
  message: Message;
  soulEmoji: string;
}

function MessageBubble({ message: m, soulEmoji }: MessageBubbleProps) {
  if (m.role === "system") {
    return (
      <div className="self-center max-w-[85%] animate-fade-slide-in">
        <div className="bg-stone-800/60 border border-stone-700 rounded-xl px-4 py-2.5 text-xs text-stone-400 font-mono whitespace-pre-wrap text-center">
          {m.text}
        </div>
        <p className="text-[10px] text-stone-700 text-center mt-1">{formatMsgTime(m.timestamp)}</p>
      </div>
    );
  }

  if (m.role === "user") {
    return (
      <div className="self-end max-w-[80%] animate-fade-slide-right">
        <div className="bg-amber-500 rounded-2xl rounded-br-none px-4 py-3 shadow-md shadow-amber-900/20">
          <p className="text-xs font-bold text-amber-950 mb-1">You</p>
          <p className="text-sm text-stone-950 whitespace-pre-wrap leading-relaxed">{m.text}</p>
        </div>
        <p className="text-[10px] text-stone-600 text-right mt-1 pr-1">{formatMsgTime(m.timestamp)}</p>
      </div>
    );
  }

  // AI message with markdown rendering
  return (
    <div className="self-start max-w-[80%] animate-fade-slide-left">
      <div className="flex items-start gap-2.5">
        <span className="text-xl mt-0.5 shrink-0">{soulEmoji}</span>
        <div>
          <div className="bg-stone-800 border border-stone-700 rounded-2xl rounded-tl-none px-4 py-3 shadow-sm">
            <p className="text-xs font-bold text-amber-400 mb-1.5">Neko AI</p>
            <div className="text-sm text-stone-200 leading-relaxed message-content">
              <Markdown content={m.text} />
            </div>
          </div>
          <p className="text-[10px] text-stone-600 mt-1 pl-1">{formatMsgTime(m.timestamp)}</p>
        </div>
      </div>
    </div>
  );
}

*/
