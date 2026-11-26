# Local PTY bridge is established prior knowledge

The user has studied multiplexer architecture (MISSION.md) and has working code that starts bash under a PTY, puts the real terminal in raw mode with restore-on-exit, forwards SIGWINCH via `pty.InheritSize`, and copies stdin/stdout both ways (`client/client.go`). Don't re-teach PTY basics; teach the *split* of that code across a socket.

**Evidence:** `client/client.go` at commit 523ec19; Unix-socket HTTP server in `server/server.go`.

**Implications:** Next skill is moving PTY ownership into the server and splitting `InheritSize` into "read size on client" / "apply size on server". Lesson 0002's quiz results (if the user shares misses) should refine this record.
