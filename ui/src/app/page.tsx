"use client";

import { useState } from "react";
import { sendChatMessage, approveCommand, rejectCommand, PendingCommand } from "@/lib/api";

type Message = { role: "user" | "ai" | "system"; text: string };

export default function Home() {
  const [messages, setMessages] = useState<Message[]>([
    { role: "system", text: "Welcome to the Neko Controller 🐾. Meow can I help you?" }
  ]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [pendingCmd, setPendingCmd] = useState<PendingCommand | null>(null);

  const sendMessage = async () => {
    if (!input.trim() || loading || pendingCmd) return;
    const msg = input;
    setInput("");
    setMessages(prev => [...prev, { role: "user", text: msg }]);
    setLoading(true);

    try {
      const res = await sendChatMessage(msg);
      setMessages(prev => [...prev, { role: "ai", text: res.reply }]);
      if (res.hasPendingCmd && res.pendingCommand) {
        setPendingCmd(res.pendingCommand);
      }
    } catch (e: any) {
      setMessages(prev => [...prev, { role: "system", text: "Error: " + e.message }]);
    }
    setLoading(false);
  };

  const handleApprove = async () => {
    if (!pendingCmd) return;
    setLoading(true);
    const cmdId = pendingCmd.id;
    setPendingCmd(null);

    try {
      const res = await approveCommand(cmdId);
      setMessages(prev => [
        ...prev, 
        { role: "system", text: `Command executed. Output:\n${res.output}` },
        { role: "ai", text: res.reply }
      ]);
    } catch (e: any) {
      setMessages(prev => [...prev, { role: "system", text: "Error executing: " + e.message }]);
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
      setMessages(prev => [...prev, { role: "system", text: "Command rejected." }]);
    } catch (e: any) {
      setMessages(prev => [...prev, { role: "system", text: "Error rejecting: " + e.message }]);
    }
    setLoading(false);
  };

  return (
    <div className="flex flex-col h-full bg-orange-50 dark:bg-stone-900 rounded-3xl shadow-xl border-4 border-amber-200 dark:border-amber-900 overflow-hidden relative">
      <div className="flex-1 p-6 overflow-y-auto flex flex-col gap-4">
        {messages.map((m, i) => (
          <div key={i} className={`p-4 rounded-2xl max-w-[80%] shadow-sm ${m.role === 'user' ? 'bg-amber-500 self-end text-white rounded-br-none' : m.role === 'system' ? 'bg-stone-200 dark:bg-stone-800 text-stone-600 dark:text-stone-400 text-sm self-center font-mono whitespace-pre-wrap rounded-xl border border-stone-300 dark:border-stone-700' : 'bg-white dark:bg-stone-800 border-2 border-amber-100 dark:border-amber-900/50 self-start text-stone-700 dark:text-stone-200 rounded-bl-none'}`}>
            {m.role === 'ai' && <div className="text-xl mb-1">🐱 Neko AI:</div>}
            {m.role === 'user' && <div className="text-xl mb-1 text-right">🧑 You:</div>}
            <p className="whitespace-pre-wrap">{m.text}</p>
          </div>
        ))}
        {loading && <div className="self-center text-amber-600 dark:text-amber-500 animate-pulse font-medium">🐾 Neko is thinking...</div>}
      </div>

      {pendingCmd && (
        <div className="p-6 bg-amber-50 dark:bg-stone-800 border-t-4 border-red-400 w-full animate-in slide-in-from-bottom">
          <p className="text-red-500 font-bold mb-2 flex items-center gap-2">🙀 Neko Wants to Execute a Command:</p>
          <div className="bg-stone-900 p-4 rounded-xl font-mono text-sm text-green-400 mb-2 overflow-x-auto shadow-inner border border-stone-700">
            {pendingCmd.command}
          </div>
          <p className="text-stone-600 dark:text-stone-300 text-sm italic mb-4">"{pendingCmd.description}"</p>
          <div className="flex gap-4">
            <button onClick={handleApprove} className="px-6 py-2 bg-green-500 hover:bg-green-400 rounded-xl text-white font-bold transition-transform active:scale-95 shadow-md">✅ Purr-fect (Approve)</button>
            <button onClick={handleReject} className="px-6 py-2 bg-red-500 hover:bg-red-400 rounded-xl text-white font-bold transition-colors shadow-md">❌ Hiss (Reject)</button>
          </div>
        </div>
      )}

      <div className="p-4 bg-orange-100 dark:bg-stone-900 border-t-2 border-amber-200 dark:border-amber-900/50 flex gap-3">
        <input 
          type="text" 
          disabled={loading || !!pendingCmd}
          className="flex-1 bg-white dark:bg-stone-800 border-2 border-transparent focus:border-amber-400 rounded-2xl px-5 py-3 focus:outline-none transition-colors disabled:opacity-50 text-stone-800 dark:text-stone-100 placeholder-stone-400 shadow-sm"
          placeholder="Ask Neko to do something... 🐟"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && sendMessage()}
        />
        <button 
          disabled={loading || !!pendingCmd || !input.trim()}
          onClick={sendMessage}
          className="bg-amber-500 hover:bg-amber-400 disabled:opacity-50 px-8 py-3 rounded-2xl font-bold text-white transition-colors shadow-md flex items-center justify-center gap-2">
            Send 🐾
        </button>
      </div>
    </div>
  );
}
