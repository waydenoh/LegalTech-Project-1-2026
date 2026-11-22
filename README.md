# Singapore Playbook Contract Reviewer (Windows Desktop App)

A standalone Windows GUI desktop application built in Go with an embedded WebView interface and local Ollama inference (`qwen3.5:4b`).

## Build & Run (Windows)

Extract into a clean folder, open PowerShell inside it, and run:
```powershell
go build -ldflags="-H=windowsgui -s -w" -o SG-Playbook-Reviewer.exe ./cmd/desktop
```
Double-click `SG-Playbook-Reviewer.exe` to open the desktop application window directly (no terminal window and no Google Chrome tabs).
