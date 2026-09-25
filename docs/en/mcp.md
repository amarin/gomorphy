# Why gomorphy Has No Built-in MCP Server

Decision: **an embedded MCP server in gomorphy will not be implemented.**
This is a deliberate choice based on an analysis of how a software agent
would use the library (building topical dictionaries, querying
wordforms/lemmas). This document records the reasoning so the decision
isn't revisited without new facts. Planning is tracked in
[todo.md](todo.md), Stage 19.

## What MCP means in this context

MCP (Model Context Protocol) is a transport between an agent and a
service: in our case, a *thin layer over `pkg/morphology`* that, instead
of CLI calls, gives the agent tool calls for:

- importing a prepared file -> `.dat`;
- dictionary queries: wordform / lemma / fuzzy search;
- (ideally) a session with an open dictionary, reused across calls.

## Why MCP doesn't save tokens

Token cost per operation is made up of **data** (the payload the agent
generates anyway) and **transport overhead** (tool wrapper, JSON, reading
the response). Estimates for the "topical dictionary" scenario:

| Operation | CLI (skill) | MCP (one at a time) | MCP (batched) |
|---|---|---|---|
| Import 15K forms (~120-180K tokens of data) | +50-100 overhead | — | +150-250 overhead |
| Import 100 forms/file | +50-100 overhead | — | +150-250 overhead |
| Check 1000 wordforms | ~60-80K overhead (one call per word) | ~150-250K overhead | ~200-300 overhead |
| Query a single form | ~50-80 overhead | ~100-200 overhead | ~100-200 overhead |

Conclusions:

1. **Import: no gain.** Tokens are almost 100% the data itself (the agent
   generates every form regardless); the wrapper adds a fixed few hundred
   tokens, which is negligible for both CLI and MCP. MCP is even slightly
   more expensive here (argument + confirmation).
2. **Queries: the gain comes from batching, not the transport.** The
   bloat comes from the CLI accepting one word per call: 1000 checks =
   1000 calls. The fix is **CLI batch mode** (`lookup -` via stdin, or
   multiple words as arguments): one call for the whole list, ~200-300
   tokens of overhead. MCP would deliver the same gain only through a
   batch tool — same payload, but at the cost of server-side machinery.
3. **Small queries.** A cosmetic difference that doesn't affect the
   decision.

## Implementation and maintenance overhead

An MCP server isn't "one more method" — it's a new subsystem:

- a new external dependency (e.g. `mark3labs/mcp-go`) in a project that
  vendors (`vendor/`);
- a separate binary/subcommand (`gomorphy mcp`), stdio transport, session
  handling;
- JSON schemas for tools and error mapping (library errors and empty
  results -> structured responses);
- transport tests, packaging, documentation — estimated at ~300-500 lines
  versus ~100 lines for a TSV importer;
- long-term burden: every import format and every new query API has to be
  duplicated in the MCP contract.

CLI batch modes that solve the same token problem are ~20-40 lines of Go,
zero dependencies, and reuse the existing `Parse`/`Lemma`/`Fuzzy`.

## What MCP would actually provide

MCP provides **UX integration**, not token savings:

- an agent in an IDE/Desktop app without shell access (Cursor, Claude
  Desktop, MCP hubs);
- a structured JSON contract instead of parsing stdout;
- a session with an open dictionary (no repeated mmap, no process
  restart).

None of these is a current priority: the target gomorphy agent works from
a terminal (the "skill + CLI" approach). The latency of restarting the
process (single- to tens-of-milliseconds) is negligible compared to the
model's response time.

## When this decision could be revisited

An MCP server is justified only if a concrete consumer appears — an agent
that **cannot** run local commands — and its value to the project
outweighs the maintenance cost. If that happens, the server must be a
thin transport over the same library primitives:

- import tools accept a file or a TSV string in the same format;
- queries are **batch tools** (`lookup_batch`, `lemmas_batch`,
  `fuzzy_batch`);
- a single source of truth — `pkg/morphology` / importers — with MCP only
  forwarding calls.

Until such a consumer appears: a deliberately minimal CLI (import today;
batch queries planned) + the planned "using the gomorphy dictionary" skill
(Stage 18) and "topical dictionary" skill (Stage 19) — neither started yet,
see [todo.md](todo.md) — remain the agent's only interface to gomorphy.
