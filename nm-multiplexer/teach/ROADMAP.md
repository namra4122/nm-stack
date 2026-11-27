# Roadmap: from local PTY bridge to detachable shell

Each milestone closes one gap from the A → B table in `reference/architecture-current-vs-target.html`. Do them in order, one per session. **You write** marks the code that holds the idea. That part is yours to write. The surrounding plumbing can be scaffolded for you.

## 1. A long-lived stream over the socket
- **Concept:** HTTP request/response vs a connection that stays open and carries bytes both ways.
- **Build:** the client opens one connection and the server echoes every byte back.
- **You write:** the choice between hijacking an HTTP connection and a raw `net.Conn`, plus the server's copy loop.
- **Check:** while you type, `lsof -p <client> -U` shows one unix socket fd held for the whole session, and typed bytes come back.

## 2. Server owns the PTY
- **Concept:** process ownership. The process holding the master decides when the shell gets SIGHUP.
- **Build:** move `pty.Start` into the server. The client only copies terminal ⇄ socket.
- **You write:** the server's two copies between the connection and the PTY master.
- **Check:** `ps -o pid,ppid,command` shows bash's PPID is the **server**. `lsof` shows `/dev/ptmx` on the server and not on the client (the reverse of Diagram A).

## 3. Framing
- **Concept:** one byte stream carrying two kinds of message, shell data and control.
- **Build:** a message format (for example type + length + payload) used in both directions.
- **You write:** the encoder and decoder.
- **Check:** a unit test round-trips payloads containing every byte value 0–255, including whatever your format treats as special.

## 4. Resize across the socket
- **Concept:** `TIOCGWINSZ` / `TIOCSWINSZ` and SIGWINCH, and which side can read the real window size.
- **Build:** a resize message, so the server sets the master's size.
- **You write:** the choice between sending rows/cols and passing the terminal fd the way tmux does (`tmux client.c:470`, `tty.c:138`).
- **Check:** resize the window, then `stty size` inside bash matches. The server's Jaeger span shows `pty.resized` with real rows/cols.

## 5. Detach and reattach
- **Concept:** the shell's lifetime is separate from the client's. SIGHUP only happens when the master closes.
- **Build:** a detach key that ends the client cleanly, and a reattach that connects to the existing PTY.
- **You write:** what the server does when reading from a client returns EOF.
- **Check:** run `sleep 1000`, then `kill -9` the client. `ps` still shows bash and sleep with PPID = server. Reattach and `jobs` still lists sleep.

## 6. Redraw on reattach
- **Concept:** screen state. Replaying raw bytes vs keeping a parsed screen like tmux's `grid` (`grid.c`, `window.c:1723`).
- **Build:** whatever your design choice needs, so a reattached client sees the current screen.
- **You write:** the choice and its buffer or screen model.
- **Check:** run `top`, detach, reattach. The current screen shows before you press any key.
