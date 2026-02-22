# MoMail

**A Modern, Lightweight FidoNet Technology Network (FTN) Mailer written in Go.**

> ⚠️ **EARLY ALPHA WARNING** ⚠️
>
> This software is currently in **Early Alpha**. It is under active development. While it is functional, it may contain bugs, incomplete features, or breaking changes.
>
> **Do not use this in a production environment without proper backups and monitoring.**

## Overview

MoMail is a fresh take on the classic FTN mailer, designed for the modern age. Built with Go, it offers a single-binary deployment, low resource usage, and a built-in web dashboard for easy monitoring. It implements the BinkP protocol to exchange mail packets and files with other nodes in FidoNet and compatible networks.

## Key Features

*   **Modern Core:** Written in pure Go (Golang) for performance and portability.
*   **BinkP 1.1 Support:** Full implementation of the BinkP protocol, including CRAM-MD5 authentication and GZIP compression (negotiated).
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
*   **Rotate logs based on `log_max_size` in config:**
    ```bash
    ./momail --cut-logs
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

# Run the container (mounting your run directory)
docker run -d \
  -p 24554:24554 \
  -p 8080:8080 \
  -v $(pwd)/run:/app/run \
  --name momail \
  momail
