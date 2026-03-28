# factop

Factorio operator.

### Rationale for using a soft mod v.s. an actual mod

* [grok answer](https://x.com/i/grok/share/9eEbNfDbw9s6PPf7qMjJNIW0l)

### Why this and not [factorio-server-manager](https://github.com/OpenFactorioServerManager/factorio-server-manager)?

* This project manages the factorio headless server in a similar manner but adds:
    * An api for applying a softmod to the currently running save, which greatly speeds up development.
        * The stop, save file changes, and start steps are handled when a new softmod is applied.
    * The factop service also exposes a [web server](https://factorio.mlctrez.com). Right now it does nothing. Future
      enhancements could be:
        * Tracking player progress, statistics, etc
        * Administrative functions like resetting the map, etc.
    * A [nats](https://docs.nats.io/nats-concepts/what-is-nats) server is embedded in the factop service.
        * The Factorio stdin, stdout, and stderr are exposed as nats subjects.
    * A rcon connection is managed by the factop service and exposed via a nats subject.
    * A mage build target for executing lua code via this rcon connection.

## Commands

All commands are sent to the factop server over [NATS](https://docs.nats.io/nats-concepts/what-is-nats) using a request/reply pattern. The `mage` build tool provides convenience wrappers for these.

### Server Management (`factop.command`)

Sent via `mage command <name>`:

| Command   | Description                                              |
|-----------|----------------------------------------------------------|
| `status`  | Returns the current server state (`running` or `stopped`) |
| `stop`    | Shuts down the Factorio server                           |
| `restart` | Restarts the Factorio server                             |
| `reset`   | Deletes the save file and restarts with a fresh map      |
| `latest`  | Starts a background download of the latest Factorio version |

### RCON (`factop.rcon`)

Sent via `mage rcon <path-to-lua-file>`. Lua scripts are stripped of comments and blank lines, prefixed with `/sc`, and executed through the managed RCON connection.

`mage prcon <delay-in-seconds> <path-to-lua-file>` does the same thing on a repeating timer.

`mage lrcon <path-to-lua-file>` sends to a local NATS server at `localhost` instead of the remote host.

### Softmod (`factorio.softmod`)

Sent via `mage softmod`. This packages the `softmod/` directory into a zip, sends it to the server, which then stops Factorio, applies the softmod to the save file, and restarts.

## Architecture

`factop` is designed as a modular service built in Go, focusing on automation and remote management of a Factorio headless server.

### Component-Based Design

The project uses the `github.com/mlctrez/bind` library for dependency injection and component lifecycle management. Key components are defined in the `service/` directory and wired together in `service/service.go`:

*   **Nats**: Embeds a NATS server and provides a central message bus for all internal and external communication.
*   **Factorio**: Manages the lifecycle of the Factorio headless server process. It handles process startup, shutdown, and redirects `stdin`, `stdout`, and `stderr` to NATS subjects.
*   **UDPBridge**: Acts as a proxy for UDP traffic on ports 4000 (incoming) and 4001 (outgoing), bridging game packets to NATS subjects (`udp.incoming`, `udp.outgoing`).
*   **Rcon**: Monitors Factorio's `stdout` for the RCON startup marker, establishes a connection, and exposes an execution interface via NATS.
*   **SoftMod**: Handles the logic for injecting a "soft mod" (Lua scripts and assets) directly into the Factorio save file (`.zip`).
*   **Command**: Subscribes to the `factop.command` subject and dispatches administrative actions to the `Factorio` component.
*   **Settings**: Ensures that necessary configuration files (server settings, map gen settings, etc.) exist in `/opt/factorio/settings`.
*   **WebServer**: A minimal HTTP server (port 8080) for health checks and future web-based management features.

### Data Flows

1.  **Command & Control**:
    *   Client (`mage`) sends a request to a NATS subject (e.g., `factop.command`).
    *   The corresponding service component (e.g., `Command`) processes the request and interacts with the `Factorio` component.
    *   A response is sent back to the client via the NATS reply subject.

2.  **Factorio I/O**:
    *   The `Factorio` component captures the server's `stdout` and `stderr` and publishes them to `factorio.stdout` and `factorio.stderr`.
    *   Subscribers to these subjects (like the `Rcon` monitor or a remote logger) can process this data in real-time.
    *   Writing to `factorio.stdin` allows sending console commands directly to the server process.

3.  **RCON Execution**:
    *   When the `Rcon` component detects the "Starting RCON interface" message in `stdout`, it connects to the local RCON port.
    *   Requests on `factop.rcon` are executed via this connection, and results are returned to the caller.

4.  **Softmod Application**:
    *   The `SoftMod` component receives a zip payload on `factorio.softmod`.
    *   It stops the Factorio server, modifies the existing save file by injecting the new files, and restarts the server. This allows for rapid iteration on Lua-based mods without manual file manipulation.

## Getting Started for Developers

### Prerequisites
*   Go 1.25+
*   [Mage](https://magefile.org/)

### Building and Deploying
The project uses `mage` for most tasks:
*   `mage service`: Builds the `factop` binary and deploys it to the configured `Host` (defined in `magefiles/mage.go`).
*   `mage softmod`: Packages the local `softmod/` directory and applies it to the remote server.

### Project Structure
*   `factop.go`: Entry point for the service.
*   `service/`: Core component logic.
*   `softmod/`: Source files for the Factorio soft mod (Lua scripts, graphics, etc.).
*   `magefiles/`: Build and automation scripts.
