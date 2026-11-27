# teach/

Everything about *learning* the multiplexer lives here. The Go code next door (`../client`, `../server`) is what you're learning *by building*.

| Path | What it is | Changes when |
| --- | --- | --- |
| `MISSION.md` | Why you're learning this and what "done" looks like. Short, a compass rather than a plan. | Your goal shifts. |
| `RESOURCES.md` | Trusted sources (man pages, tmux source, articles). Lessons cite these. | A better source turns up. |
| `lessons/NNNN-*.html` | One small lesson per file, numbered in order. Open in a browser. | A new concept is taught. |
| `reference/*.html` | Cheat sheets and diagrams compressed from lessons. Quick lookup. | A lesson adds something worth keeping. |
| `learning-records/NNNN-*.md` | What you've shown you know, like an ADR for your brain. Used to pick the next lesson. | You demonstrate or revise understanding. |
| `ROADMAP.md` | Milestones in dependency order, each with an acceptance check. | A milestone is done or re-scoped. |
| `TODO.md` | Build checklist for the Go code. | Code lands. |
| `NOTES.md` | Scratchpad: your preferences and open questions for the teacher. | Any session. |
| `assets/` | Shared lesson parts: `lesson.css`, `quiz.js`, `diagram.js` (renders Mermaid). | A lesson needs a reusable part. |

## Running `/teach`

The teach skill reads `MISSION.md`, `lessons/` and the rest from the **directory it is run in**. Start it from this folder (`cd teach`). Run it from the repo root and it will find no mission and start a fresh workspace there.
