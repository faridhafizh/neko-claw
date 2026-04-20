import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import Link from "next/link";

const inter = Inter({ subsets: ["latin"], variable: "--font-inter" });

export const metadata: Metadata = {
  title: "Neko AI Controller 🐾",
  description: "Control your computer with a pawsome AI agent",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className={inter.variable}>
      <body className="bg-stone-950 text-stone-100 antialiased min-h-screen flex flex-col font-sans">
        <header className="bg-stone-900/80 backdrop-blur-md border-b border-stone-800 shrink-0 shadow-lg sticky top-0 z-50">
          <div className="max-w-7xl mx-auto px-4 py-3 flex items-center justify-between">
            <Link href="/" className="flex items-center gap-2 group">
              <span className="text-2xl group-hover:animate-bounce transition-all">🐱</span>
              <h1 className="text-xl font-black bg-gradient-to-r from-orange-400 via-amber-400 to-yellow-400 bg-clip-text text-transparent">
                Neko AI Controller
              </h1>
            </Link>
            <nav className="flex gap-1">
              <Link
                href="/"
                className="flex items-center gap-1.5 px-3 py-2 rounded-xl text-sm font-semibold text-stone-400 hover:text-amber-400 hover:bg-stone-800 transition-all"
              >
                🏠 Chat
              </Link>
              <Link
                href="/memory"
                className="flex items-center gap-1.5 px-3 py-2 rounded-xl text-sm font-semibold text-stone-400 hover:text-amber-400 hover:bg-stone-800 transition-all"
              >
                🧠 Memory
              </Link>
              <Link
                href="/files"
                className="flex items-center gap-1.5 px-3 py-2 rounded-xl text-sm font-semibold text-stone-400 hover:text-amber-400 hover:bg-stone-800 transition-all"
              >
                📂 Files
              </Link>
              <Link
                href="/settings"
                className="flex items-center gap-1.5 px-3 py-2 rounded-xl text-sm font-semibold text-stone-400 hover:text-amber-400 hover:bg-stone-800 transition-all"
              >
                ⚙️ Settings
              </Link>
            </nav>
          </div>
        </header>
        <main className="flex-1 overflow-hidden w-full">
          {children}
        </main>
      </body>
    </html>
  );
}
