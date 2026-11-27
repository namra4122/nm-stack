# NM Multiplexer

1. Understand how a multiplexer works internally. Study [tmux](https://github.com/tmux/tmux) and [Herdr](https://github.com/herdrdev/herdr).
   - [x] tmux client/server split: fd passing, resize, pane PTY, grid (see `reference/architecture-current-vs-target.html`)
   - [ ] Herdr: not reviewed yet

2. Build the individual components:
   - [x] Basic IPC server and client over a Unix socket (HTTP `/health`, `/memory`)
   - [x] Local PTY shell: bash on a PTY, raw mode, resize, two-way copy (`client/client.go`, commit c51af33)
   - [x] Tracing client and server to Jaeger
   - [ ] Move PTY ownership into the server; client streams keystrokes and output over the socket
   - [ ] Resize across the socket
   - [ ] Detach and reattach without killing the shell
   - [ ] Redraw the screen on reattach
