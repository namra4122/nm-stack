# Notes

- Prefers quick revision passes before building a new component (2026-11-26: asked for a multiplexer recap before writing the server).
- Likes a tidy workspace; remove scratch/runtime files (`debug.log`, `*.sock`, duplicate entry points) as part of sessions. Runtime artifacts are gitignored.
- TODO.md mentions Herdr (github.com/herdrdev/herdr) as a second reference implementation alongside tmux — not yet reviewed or added to RESOURCES.md.
- Verified 2026-11-27 (tmux a6ce3d0): the server reads pane output (`window.c:1723`) into `struct grid` (`grid.c`), and redraw only sets flags (`server-fn.c:59`). The client passes its stdin fd to the server over SCM_RIGHTS (`client.c:470-473`). `MSG_RESIZE` has no payload; the server calls `TIOCGWINSZ` itself (`server-client.c:2833`, `tty.c:138`). This contradicts the cheat sheet's "client reports size". Treat it as a design choice for the user, not an error.
- Workspace moved to `teach/` on 2026-11-27. Run /teach from `teach/`.
- Prefers diagrams built from live evidence (ps/lsof/Jaeger) over diagrams from memory.
