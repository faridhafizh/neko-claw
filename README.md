# 🐱 Neko-Claw AI Controller

Neko-Claw is a powerful and "pawsome" computer control superapp that allows you to interact with your Windows machine using AI. Featuring a sleek cat-themed UI and powered by Zhipu AI (GLM-4), it provides a safe yet efficient way to automate tasks through PowerShell commands.

![Neko-Claw UI](.qwen/screenshot_placeholder.png) *Note: Aesthetic cat-themed interface with amber and stone color palette.*

## 🐾 Features

- **Cat-Themed Interface**: A warm, user-friendly UI designed with a premium "Neko" aesthetic.
- **Human-in-the-Loop Safety**: AI proposes PowerShell commands, but nothing runs without your explicit "Purr-fect" (Approve) or "Hiss" (Reject).
- **Zhipu AI Integration**: Optimized for `glm-4.7-flash` model out of the box.
- **One-Command Start**: Intelligent backend that automatically builds the Next.js frontend if needed.
- **Smart Output Cleaning**: Automatically strips ANSI escape codes from terminal outputs for clean readability.
- **Flexible Configuration**: Easily change API Keys, Models, and URLs directly from the Settings menu.

## 🛠️ Tech Stack

- **Backend**: Go (Golang)
- **Frontend**: Next.js (TypeScript, Tailwind CSS)
- **AI Integration**: OpenAI-compatible API (Zhipu AI Recommended)

## 🚀 Getting Started

### Prerequisites

- [Go](https://go.dev/dl/) 1.20+
- [Node.js & npm](https://nodejs.org/) (for building the UI)
- [PowerShell/pwsh](https://learn.microsoft.com/en-us/powershell/scripting/install/installing-powershell) (standard on Windows)

### Installation & Running

1. **Clone the repository**:
   ```bash
   git clone https://github.com/yourusername/neko-claw.git
   cd neko-claw
   ```

2. **Run the application**:
   Navigate to the `agent` folder and run the Go server. It will automatically install UI dependencies and build the frontend on the first run.
   ```bash
   cd agent
   go run .
   ```

3. **Access the App**:
   Open your browser and go to `http://localhost:8080`.

## ⚙️ Configuration

Once the app is running:
1. Click on **⚙️ Settings** in the top navigation bar.
2. Enter your **Zhipu AI API Key** (get it from [BigModel.cn](https://open.bigmodel.cn/)).
3. Ensure the Endpoint is set to: `https://open.bigmodel.cn/api/paas/v4/`
4. Save the configuration and start chatting with Neko!

## ⚠️ Safety Warning

This application allows an AI to suggest commands that can modify your system. **Always review the commands in the approval box before clicking "Purr-fect"**.

---

*Made with 🐾 and AI.*
