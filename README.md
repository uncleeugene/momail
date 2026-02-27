# MoMail

**A Modern, Lightweight FidoNet Technology Network (FTN) Mailer written in Go.**

> ⚠️ **EARLY ALPHA WARNING** ⚠️
>
> This software is currently in **Early Alpha**. It is under active development. While it is functional, it may and it will contain bugs, incomplete features, or breaking changes.
>
> **Do not use this in a production environment without proper backups and monitoring.**

## Overview

MoMail is a fresh take on the classic FTN mailer, designed for the modern age. Built with Go, it offers a single-binary deployment, low resource usage, and a built-in web dashboard for easy monitoring. It implements the BinkP protocol to exchange mail packets and files with other nodes in FidoNet and compatible networks.

## Key Features

*   **Modern Core:** Written in pure Go (Golang) for performance and portability.
*   **BinkP 1.0 Support:** Full implementation of the BinkP protocol, including CRAM-MD5 authentication. (Not fully completed yet)
*   **Smart Routing:**
    *   Standard Binkley-style outbound directory structure (BSO).
    *   Supports 4D addressing (Zone:Net/Node.Point).
    *   Automatic outbound scanning using file system notifications.
*   **Flexible Lookup:**
    *   DNS-based address resolution (binkp.net).
    *   Legacy Nodelist support (compiles and uses raw nodelists).
*   **Automation & Scripting:**
    *   **Cron Scheduler:** Built-in cron-like scheduler to run maintenance scripts or polls.
    *   **File Triggers:** Execute external commands automatically when specific files (like `*.pkt` or `*.tic`) are received.
*   **Observability:**
    *   **Web Dashboard:** A beautiful, real-time web interface to monitor active sessions, traffic, and queues.
    *   **Live Logs:** WebSocket-based log streaming directly to your browser.
    *   **HTTP API:** Full control over the mailer via a RESTful API.
*   **Security:**
    *   Secure/Insecure inbound separation.
    *   Option to refuse unauthenticated traffic.
*   **Easy Deployment:**
    *   Docker-ready (Dockerfile included).
    *   Single YAML configuration file.

## Getting Started

### Prerequisites

*   Go 1.23 or higher (if building from source)
*   A valid FidoNet address and uplink.

### Installation

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/yourusername/momail.git
    cd momail
    ```

2.  **Build the binary:**
    ```bash
    go build -o momail .
    ```

3.  **Configure:**
    Copy the distribution config and edit it with your details.
    ```bash
    mkdir -p run/config run/logs run/inbound run/outbound
    cp config.yaml.dist run/config/config.yaml
    nano run/config/config.yaml
    ```

4.  **Run:**
    ```bash
    ./momail -c run/config/config.yaml
    ```

### Usage
#### Command line interface

MoMail can be run as a daemon to handle mail continuously, or as a one-off command for specific tasks.

*   **Run as a daemon:**
    ```bash
    # The -w flag enables automatic config reloading on change.
    ./momail -c ./run/config/config.yaml -w
    ```

*   **Poll a node immediately (dials the node right away):**
    ```bash
    ./momail -p 2:5020/828
    ```

*   **Queue a poll (creates a flow file for the scheduler to pick up):**
    ```bash
    # Normal poll (.flo)
    ./momail -q 2:5020/828
    # Crash poll (.clo, highest priority)
    ./momail -q 2:5020/828 --crash
    ```

#### Dashboard
MoMail includes a real-time web dashboard for monitoring sessions, queues, and logs.

1.  **Open in Browser:** `http://localhost:8080/dashboard`
2.  **Authentication:** If you set an `api_token` in `config.yaml`, append it to the URL:
    `http://localhost:8080/dashboard?token=YOUR_SECRET_TOKEN`

The dashboard allows you to:
*   Monitor active BinkP sessions and traffic.
*   View and manage the outbound mail queue.
*   Watch live logs and control the mailer (scan, mute, etc).

### Docker Support

MoMail is designed to run in a container.

```bash
# Build the image
docker build -t momail .
   ```
#### Run the container (mounting your run directory)
```bash
docker run -d \
  -p 24554:24554 \
  -p 8080:8080 \
  -v $(pwd)/run:/app/run \
  --name momail \
  momail
  ```

## Cross-Platform Notes

MoMail is designed to be cross-platform and runs on Linux, macOS, and Windows.

However, when defining `command` for `triggers` or `tasks` in your `config.yaml`, be aware that these commands are executed by the operating system's default shell.

*   On **Linux/macOS** and other Unix-like systems, commands are executed via `sh -c "..."`.
*   On **Windows**, commands are executed via `cmd /C "..."`.

You must write your commands to be compatible with the target operating system's shell. For example, a simple file copy would be `cp source dest` on Linux but `copy source dest` on Windows.

For convenience, the `{file}` placeholder in trigger commands will always have its path separators converted to forward slashes (`/`), which is compatible with both `sh` and modern `cmd.exe`/PowerShell.

### Configuration Reloading

MoMail supports reloading its configuration without restarting the daemon. The method depends on your operating system.

*   On **Linux/macOS**, you can send the `SIGHUP` signal to the `momail` process to trigger a configuration reload.
    ```bash
    pkill -HUP momail
    ```

*   On **Windows**, signals like `SIGHUP` are not available. You can trigger a reload in two ways:
    1.  **File Watching:** If you start the daemon with the `-w` or `--watch-config` flag, it will automatically reload when `config.yaml` is saved.
    2.  **API Call:** You can send a request to the built-in HTTP API. This is useful for scripting or manual reloads. You can use `curl` (included in modern Windows) or PowerShell.

        **Using `curl` (from Command Prompt):**
        ```cmd
        curl -X POST http://localhost:8080/api/control -H "Content-Type: application/json" -d "{\"command\": \"reload\"}"
        ```

        If you have set an `api_token` in your config, you must provide it as a Bearer token in the header:
        ```cmd
        curl -X POST http://localhost:8080/api/control -H "Authorization: Bearer YOUR_SECRET_TOKEN" -H "Content-Type: application/json" -d "{\"command\": \"reload\"}"
        ```

        **Using PowerShell:**
        ```powershell
        Invoke-RestMethod -Uri http://localhost:8080/api/control -Method Post -Body '{"command": "reload"}' -ContentType 'application/json'
        # With an API token:
        # Invoke-RestMethod -Uri http://localhost:8080/api/control -Method Post -Body '{"command": "reload"}' -ContentType 'application/json' -Headers @{"Authorization"="Bearer YOUR_SECRET_TOKEN"}
        ```
