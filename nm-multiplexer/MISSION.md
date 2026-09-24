# Mission: Build a terminal multiplexer in Go

## Why
I have studied how a multiplexer works and want to build my own. The goal is a working client/server program that keeps an interactive shell alive independently of its attached terminal.

## Success looks like
- Start a server that owns a PTY-backed shell and attach a terminal client over a Unix socket.
- Send keyboard input to the shell and stream output back to the client reliably.
- Detach and reattach without terminating the shell; handle terminal resize and clean shutdown.

## Constraints
- The existing project is Go and already depends on `creack/pty` and `x/term`.
- Build incrementally from the current implementation; teach one focused concept at a time.

## Out of scope
- Full tmux feature parity (multiple sessions, windows, panes, layout management, and configuration) until the single-session vertical slice works.
