"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { fetchSettings, saveSettings, Settings, fetchSouls, setActiveSoul, SoulProfile } from "@/lib/api";

export default function SettingsPage() {
  const router = useRouter();
  const [settings, setSettings] = useState<Settings>({ apiKey: "", model: "glm-4.7-flash", apiUrl: "https://open.bigmodel.cn/api/paas/v4/" });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  
  const [souls, setSouls] = useState<Record<string, SoulProfile>>({});
  const [activeSoulId, setActiveSoulId] = useState("default");
  const [soulsLoading, setSoulsLoading] = useState(true);

  useEffect(() => {
    fetchSettings().then(s => {
      if (s) setSettings(s);
      setLoading(false);
    }).catch(e => {
        setLoading(false);
    });
    
    fetchSouls().then(res => {
      setSouls(res.souls);
      setActiveSoulId(res.activeSoul);
      setSoulsLoading(false);
    }).catch(e => {
      console.error("Failed to load souls:", e);
      setSoulsLoading(false);
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

  const handleSoulChange = async (soulId: string) => {
    try {
      await setActiveSoul(soulId);
      setActiveSoulId(soulId);
    } catch (e: any) {
      alert("Failed to change soul: " + e.message);
    }
  };

  if (loading) return <div className="p-8 text-center text-slate-400">Loading settings...</div>;

  return (
    <div className="max-w-4xl mx-auto p-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-3xl font-bold text-stone-800 dark:text-stone-100 flex items-center gap-2">
          ⚙️ Settings
        </h1>
        <button
          onClick={() => router.push("/")}
          className="px-4 py-2 bg-amber-500 hover:bg-amber-400 text-white rounded-xl font-medium transition-colors"
        >
          ← Back to Chat
        </button>
      </div>

      {/* Soul Selection */}
      <div className="bg-white dark:bg-stone-800 border-2 border-amber-200 dark:border-amber-900/50 rounded-xl p-6 mb-6 shadow-md">
        <h2 className="text-xl font-bold mb-4 text-stone-700 dark:text-stone-200 flex items-center gap-2">
          🎭 Neko Soul (Personality)
        </h2>
        {soulsLoading ? (
          <div className="text-center py-4 text-amber-600 dark:text-amber-500 animate-pulse">Loading souls...</div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {Object.entries(souls).map(([id, soul]) => (
              <button
                key={id}
                onClick={() => handleSoulChange(id)}
                className={`p-4 rounded-xl border-2 transition-all text-left ${
                  activeSoulId === id
                    ? "border-amber-500 bg-amber-50 dark:bg-amber-900/20 shadow-md"
                    : "border-stone-200 dark:border-stone-600 bg-stone-50 dark:bg-stone-700 hover:border-amber-300 dark:hover:border-amber-700"
                }`}
              >
                <div className="text-3xl mb-2">{soul.emoji}</div>
                <div className="font-semibold text-stone-800 dark:text-stone-100">{soul.name}</div>
                <div className="text-sm text-stone-600 dark:text-stone-400 mt-1">{soul.description}</div>
              </button>
            ))}
          </div>
        )}
      </div>

      {/* AI Provider Settings */}
      <div className="bg-white dark:bg-stone-800 border-2 border-amber-200 dark:border-amber-900/50 rounded-xl p-6 shadow-md">
        <h2 className="text-xl font-bold mb-4 text-stone-700 dark:text-stone-200 flex items-center gap-2">
          🐾 AI Provider Configuration
        </h2>

        <form onSubmit={handleSave} className="flex flex-col gap-6">
          <div>
            <label className="block text-sm font-medium mb-2 text-stone-600 dark:text-stone-400">API Provider / URL endpoint</label>
            <input
              type="text"
              value={settings.apiUrl}
              onChange={e => setSettings({...settings, apiUrl: e.target.value})}
              className="w-full bg-stone-50 dark:bg-stone-700 border-2 border-stone-300 dark:border-stone-600 rounded-lg px-4 py-2 focus:outline-none focus:border-amber-500 text-stone-800 dark:text-stone-100"
              placeholder="https://open.bigmodel.cn/api/paas/v4/"
            />
            <p className="text-xs text-stone-500 dark:text-stone-400 mt-1">For Zhipu AI, use the compatible endpoint. Leave blank for default OpenAI.</p>
          </div>

          <div>
            <label className="block text-sm font-medium mb-2 text-stone-600 dark:text-stone-400">API Key</label>
            <input
              type="password"
              value={settings.apiKey}
              onChange={e => setSettings({...settings, apiKey: e.target.value})}
              className="w-full bg-stone-50 dark:bg-stone-700 border-2 border-stone-300 dark:border-stone-600 rounded-lg px-4 py-2 focus:outline-none focus:border-amber-500 text-stone-800 dark:text-stone-100"
              placeholder="sk-..."
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-2 text-stone-600 dark:text-stone-400">Model Name</label>
            <input
              type="text"
              value={settings.model}
              onChange={e => setSettings({...settings, model: e.target.value})}
              className="w-full bg-stone-50 dark:bg-stone-700 border-2 border-stone-300 dark:border-stone-600 rounded-lg px-4 py-2 focus:outline-none focus:border-amber-500 text-stone-800 dark:text-stone-100"
              placeholder="glm-4.7-flash"
            />
          </div>

          <div className="flex items-center gap-4 mt-4">
            <button
              type="submit"
              disabled={saving}
              className="bg-amber-500 hover:bg-amber-400 disabled:opacity-50 text-white px-6 py-2 rounded-lg font-semibold transition-colors shadow-md"
            >
              {saving ? "Saving..." : "Save Configuration"}
            </button>
            {saved && <span className="text-green-600 dark:text-green-400 animate-pulse">✓ Saved successfully!</span>}
          </div>
        </form>
      </div>
    </div>
  );
}
