"use client";

import { useState, useEffect } from "react";
import { fetchSettings, saveSettings, Settings } from "@/lib/api";

export default function SettingsPage() {
  const [settings, setSettings] = useState<Settings>({ apiKey: "", model: "glm-4.7-flash", apiUrl: "https://open.bigmodel.cn/api/paas/v4/" });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    fetchSettings().then(s => {
      if (s) setSettings(s);
      setLoading(false);
    }).catch(e => {
        setLoading(false);
    });
  }, []);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setSaved(false);
    try {
      await saveSettings(settings);
      setSaved(true);
      setTimeout(() => setSaved(false), 3000);
    } catch(e) {
      alert("Failed to save");
    }
    setSaving(false);
  };

  if (loading) return <div className="p-8 text-center text-slate-400">Loading settings...</div>;

  return (
    <div className="max-w-2xl mx-auto bg-slate-900 border border-slate-800 rounded-xl p-8 mt-10">
      <h2 className="text-2xl font-bold mb-6 text-amber-500 flex items-center gap-2">🐾 AI Provider Configuration</h2>
      
      <form onSubmit={handleSave} className="flex flex-col gap-6">
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-2">API Provider / URL endpoint</label>
          <input 
            type="text" 
            value={settings.apiUrl} 
            onChange={e => setSettings({...settings, apiUrl: e.target.value})}
            className="w-full bg-slate-800 border border-amber-900/50 rounded-lg px-4 py-2 focus:outline-none focus:border-amber-500 text-slate-100"
            placeholder="https://open.bigmodel.cn/api/paas/v4/"
          />
          <p className="text-xs text-slate-400 mt-1">For Zhipu AI, use the compatible endpoint. Leave blank for default OpenAI.</p>
        </div>

        <div>
          <label className="block text-sm font-medium text-slate-300 mb-2">API Key</label>
          <input 
            type="password" 
            value={settings.apiKey} 
            onChange={e => setSettings({...settings, apiKey: e.target.value})}
            className="w-full bg-slate-800 border border-amber-900/50 rounded-lg px-4 py-2 focus:outline-none focus:border-amber-500 text-slate-100"
            placeholder="sk-..."
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-slate-300 mb-2">Model Name</label>
          <input 
            type="text" 
            value={settings.model} 
            onChange={e => setSettings({...settings, model: e.target.value})}
            className="w-full bg-slate-800 border border-amber-900/50 rounded-lg px-4 py-2 focus:outline-none focus:border-amber-500 text-slate-100"
            placeholder="glm-4.7-flash"
          />
        </div>

        <div className="flex items-center gap-4 mt-4">
          <button 
            type="submit" 
            disabled={saving}
            className="bg-amber-600 hover:bg-amber-500 text-white disabled:opacity-50 px-6 py-2 rounded-lg font-semibold transition-colors shadow-lg shadow-amber-900/20"
          >
            {saving ? "Saving..." : "Save Configuration"}
          </button>
          {saved && <span className="text-green-500 animate-in fade-in">Saved successfully!</span>}
        </div>
      </form>
    </div>
  );
}
