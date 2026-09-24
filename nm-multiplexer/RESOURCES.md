# Go Terminal Multiplexer Resources

## Knowledge

- [Go `os/exec` package](https://pkg.go.dev/os/exec)
  Official process lifecycle API, including `Start` and `Wait`; use when managing the shell subprocess.
- [`creack/pty`](https://github.com/creack/pty)
  PTY allocation and terminal attachment behavior; use when building the server-owned interactive shell.
- [Go `net` package](https://pkg.go.dev/net)
  Official Unix-domain stream socket support; use when connecting the client and server.
- [tmux(1) manual](https://man7.org/linux/man-pages/man1/tmux.1.html)
  Describes the user-facing session/client model and detach/attach behavior; use for product-level expectations, not as a requirement to clone tmux.
- [tmux source repository](https://github.com/tmux/tmux)
  Reference implementation for mature client/server architecture; consult only for specific design questions after the small vertical slice works.

## Wisdom (Communities)

- No community preference established. Defer community suggestions until a concrete design or debugging question benefits from practitioner feedback.
