"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { fetchMemories, addMemory, deleteMemory, Memory, fetchMemoryStats } from "@/lib/api";

export default function MemoryPage() {
  const router = useRouter();
  const [memories, setMemories] = useState<Memory[]>([]);
  const [loading, setLoading] = useState(true);
  const [category, setCategory] = useState("");
  const [newContent, setNewContent] = useState("");
  const [newCategory, setNewCategory] = useState("facts");
  const [newPriority, setNewPriority] = useState(3);
  const [newTags, setNewTags] = useState("");
  const [stats, setStats] = useState<{total: number, byCategory: Record<string, number>} | null>(null);

  useEffect(() => {
    loadMemories();
    loadStats();
  }, [category]);

  const loadMemories = async () => {
    try {
      setLoading(true);
      const res = await fetchMemories(category || undefined);
      setMemories(res.memories || []);
    } catch (e: any) {
      console.error("Failed to load memories:", e);
    } finally {
      setLoading(false);
    }
  };

  const loadStats = async () => {
    try {
      const res = await fetchMemoryStats();
      setStats(res);
    } catch (e: any) {
      console.error("Failed to load stats:", e);
    }
  };

  const handleAddMemory = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newContent.trim()) return;

    try {
      const tags = newTags.split(",").map(t => t.trim()).filter(t => t);
      await addMemory(newContent, newCategory, newPriority, tags);
      setNewContent("");
      setNewTags("");
      loadMemories();
      loadStats();
    } catch (e: any) {
      console.error("Failed to add memory:", e);
      alert("Failed to add memory: " + e.message);
    }
  };

  const handleDeleteMemory = async (id: string) => {
    if (!confirm("Delete this memory?")) return;

    try {
      await deleteMemory(id);
      loadMemories();
      loadStats();
    } catch (e: any) {
      console.error("Failed to delete memory:", e);
      alert("Failed to delete memory: " + e.message);
    }
  };

  const getCategoryColor = (cat: string) => {
    switch (cat) {
      case "facts": return "bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200";
      case "preferences": return "bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200";
      case "events": return "bg-purple-100 dark:bg-purple-900 text-purple-800 dark:text-purple-200";
      case "commands": return "bg-orange-100 dark:bg-orange-900 text-orange-800 dark:text-orange-200";
      default: return "bg-stone-100 dark:bg-stone-800 text-stone-800 dark:text-stone-200";
    }
  };

  return (
    <div className="min-h-screen bg-orange-50 dark:bg-stone-900 p-6">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-3xl font-bold text-stone-800 dark:text-stone-100 flex items-center gap-2">
            🧠 Neko Memory Bank
          </h1>
          <button
            onClick={() => router.push("/")}
            className="px-4 py-2 bg-amber-500 hover:bg-amber-400 text-white rounded-xl font-medium transition-colors"
          >
            ← Back to Chat
          </button>
        </div>

        {/* Stats */}
        {stats && (
          <div className="bg-white dark:bg-stone-800 rounded-2xl p-4 mb-6 shadow-md border-2 border-amber-200 dark:border-amber-900/50">
            <h2 className="text-lg font-semibold mb-3 text-stone-700 dark:text-stone-200">Memory Statistics</h2>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
              <div className="bg-amber-50 dark:bg-stone-700 p-3 rounded-xl">
                <div className="text-2xl font-bold text-amber-600 dark:text-amber-400">{stats.total}</div>
                <div className="text-sm text-stone-600 dark:text-stone-400">Total Memories</div>
              </div>
              {Object.entries(stats.byCategory).map(([cat, count]) => (
                <div key={cat} className="bg-amber-50 dark:bg-stone-700 p-3 rounded-xl">
                  <div className="text-2xl font-bold text-amber-600 dark:text-amber-400">{count}</div>
                  <div className="text-sm text-stone-600 dark:text-stone-400 capitalize">{cat}</div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Add Memory Form */}
        <div className="bg-white dark:bg-stone-800 rounded-2xl p-6 mb-6 shadow-md border-2 border-amber-200 dark:border-amber-900/50">
          <h2 className="text-xl font-semibold mb-4 text-stone-700 dark:text-stone-200">Add New Memory</h2>
          <form onSubmit={handleAddMemory} className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-2 text-stone-600 dark:text-stone-400">Content</label>
              <textarea
                value={newContent}
                onChange={(e) => setNewContent(e.target.value)}
                className="w-full p-3 border-2 border-stone-300 dark:border-stone-600 rounded-xl bg-stone-50 dark:bg-stone-700 text-stone-800 dark:text-stone-100"
                rows={3}
                placeholder="What should Neko remember?"
              />
            </div>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label className="block text-sm font-medium mb-2 text-stone-600 dark:text-stone-400">Category</label>
                <select
                  value={newCategory}
                  onChange={(e) => setNewCategory(e.target.value)}
                  className="w-full p-3 border-2 border-stone-300 dark:border-stone-600 rounded-xl bg-stone-50 dark:bg-stone-700 text-stone-800 dark:text-stone-100"
                >
                  <option value="facts">Facts</option>
                  <option value="preferences">Preferences</option>
                  <option value="events">Events</option>
                  <option value="commands">Commands</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium mb-2 text-stone-600 dark:text-stone-400">Priority (1-5)</label>
                <input
                  type="number"
                  min="1"
                  max="5"
                  value={newPriority}
                  onChange={(e) => setNewPriority(parseInt(e.target.value))}
                  className="w-full p-3 border-2 border-stone-300 dark:border-stone-600 rounded-xl bg-stone-50 dark:bg-stone-700 text-stone-800 dark:text-stone-100"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2 text-stone-600 dark:text-stone-400">Tags (comma-separated)</label>
                <input
                  type="text"
                  value={newTags}
                  onChange={(e) => setNewTags(e.target.value)}
                  className="w-full p-3 border-2 border-stone-300 dark:border-stone-600 rounded-xl bg-stone-50 dark:bg-stone-700 text-stone-800 dark:text-stone-100"
                  placeholder="tag1, tag2"
                />
              </div>
            </div>
            <button
              type="submit"
              className="w-full bg-amber-500 hover:bg-amber-400 text-white font-bold py-3 rounded-xl transition-colors"
            >
              🐾 Add Memory
            </button>
          </form>
        </div>

        {/* Filter */}
        <div className="bg-white dark:bg-stone-800 rounded-2xl p-4 mb-6 shadow-md border-2 border-amber-200 dark:border-amber-900/50">
          <label className="block text-sm font-medium mb-2 text-stone-600 dark:text-stone-400">Filter by Category</label>
          <div className="flex gap-2 flex-wrap">
            <button
              onClick={() => setCategory("")}
              className={`px-4 py-2 rounded-xl font-medium transition-colors ${
                !category 
                  ? "bg-amber-500 text-white" 
                  : "bg-stone-100 dark:bg-stone-700 text-stone-600 dark:text-stone-300 hover:bg-stone-200 dark:hover:bg-stone-600"
              }`}
            >
              All
            </button>
            {["facts", "preferences", "events", "commands"].map((cat) => (
              <button
                key={cat}
                onClick={() => setCategory(cat)}
                className={`px-4 py-2 rounded-xl font-medium transition-colors capitalize ${
                  category === cat
                    ? "bg-amber-500 text-white"
                    : "bg-stone-100 dark:bg-stone-700 text-stone-600 dark:text-stone-300 hover:bg-stone-200 dark:hover:bg-stone-600"
                }`}
              >
                {cat}
              </button>
            ))}
          </div>
        </div>

        {/* Memories List */}
        <div className="bg-white dark:bg-stone-800 rounded-2xl p-6 shadow-md border-2 border-amber-200 dark:border-amber-900/50">
          <h2 className="text-xl font-semibold mb-4 text-stone-700 dark:text-stone-200">
            Memories ({memories.length})
          </h2>
          {loading ? (
            <div className="text-center py-8 text-amber-600 dark:text-amber-500 animate-pulse">Loading memories...</div>
          ) : memories.length === 0 ? (
            <div className="text-center py-8 text-stone-500 dark:text-stone-400">No memories found. Add one above!</div>
          ) : (
            <div className="space-y-3">
              {memories.map((mem) => (
                <div
                  key={mem.id}
                  className="p-4 bg-stone-50 dark:bg-stone-700 rounded-xl border border-stone-200 dark:border-stone-600 hover:shadow-md transition-shadow"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex-1">
                      <p className="text-stone-800 dark:text-stone-100 mb-2">{mem.content}</p>
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className={`text-xs px-2 py-1 rounded-full font-medium capitalize ${getCategoryColor(mem.category)}`}>
                          {mem.category}
                        </span>
                        <span className="text-xs text-stone-500 dark:text-stone-400">Priority: {mem.priority}/5</span>
                        {mem.tags && mem.tags.length > 0 && (
                          <span className="text-xs text-stone-500 dark:text-stone-400">Tags: {mem.tags.join(", ")}</span>
                        )}
                      </div>
                    </div>
                    <button
                      onClick={() => handleDeleteMemory(mem.id)}
                      className="px-3 py-1 bg-red-500 hover:bg-red-400 text-white text-sm rounded-lg transition-colors"
                    >
                      Delete
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
