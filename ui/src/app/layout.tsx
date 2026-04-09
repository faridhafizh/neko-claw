import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import Link from "next/link";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "Neko AI Controller 🐾",
  description: "Control your computer with a pawsome AI",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className={`${inter.className} bg-stone-100 dark:bg-stone-950 text-stone-800 dark:text-stone-100 antialiased min-h-screen flex flex-col`}>
        <header className="bg-white dark:bg-stone-900 border-b-4 border-amber-300 dark:border-amber-700 shrink-0 shadow-sm">
          <div className="max-w-5xl mx-auto px-4 py-4 flex items-center justify-between">
            <h1 className="text-2xl font-black bg-gradient-to-r from-orange-500 to-amber-500 bg-clip-text text-transparent flex items-center gap-2">
              🐱 Neko AI Controller
            </h1>
            <nav className="flex gap-6 font-bold text-stone-600 dark:text-stone-300">
              <Link href="/" className="hover:text-amber-500 transition-colors flex items-center gap-1">🏠 Dashboard</Link>
              <Link href="/settings" className="hover:text-amber-500 transition-colors flex items-center gap-1">⚙️ Settings</Link>
            </nav>
          </div>
        </header>
        <main className="flex-1 overflow-auto max-w-5xl mx-auto w-full p-4">
          {children}
        </main>
      </body>
    </html>
  );
}
