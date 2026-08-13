# Workbench Knowledge System

Workbench should become more useful over time rather than restarting from zero whenever a chat context fills up or a new project begins.

The knowledge system has four layers:

1. **Conversation compaction** — distil a long working session into a bounded context capsule containing current objective, verified state, decisions, constraints, open threads and durable references.
2. **Project memory** — facts and decisions that belong to one repository: architecture choices, conventions, commands, known traps, accepted trade-offs and project-specific procedures.
3. **Global memory** — reusable knowledge that applies across projects: operator preferences, environment-independent engineering patterns, tool behaviour and durable policies.
4. **Routine/library memory** — reusable recipes and implementation assets: tested command sequences, patches/templates, migration procedures, code snippets and small components that should be retrieved before rebuilding the same thing.

## Design rules

- Memory is structured, scoped and attributable; it is not one giant transcript summary.
- Every durable item has a scope, kind, source/provenance, timestamps and optional tags.
- Project memory never silently becomes global memory.
- Secret-like material is refused from model-readable memory and belongs in the vault instead.
- Compaction keeps a short active context capsule plus references to durable memories rather than copying the whole history forward.
- Retrieval is relevance-first and budgeted: the lead AI asks for the small set of memories/routines most likely to help the current task.
- Routines should be versionable and testable. A successful routine can be promoted from project scope to global scope explicitly.
- Generated code should be searched for reusable prior implementations before a new version is created.
- Autonomous workers receive a bounded advisory slice of the latest project context plus relevant project/global memory before they start. Repository state remains authoritative, so stale memory cannot silently override the current tree.

## Context capsule

A context capsule is the handoff object used when a conversation is getting long. It should contain:

- current outcome;
- current repository/project identity;
- verified completed work;
- decisions and constraints that still matter;
- active task IDs / branches / artefact references;
- unresolved questions;
- memory and routine IDs relevant to continuation;
- a bounded next-action summary.

The capsule is designed to be small enough to inject into a fresh conversation or worker prompt without replaying the original transcript.

## Retrieval ladder

Before implementing a non-trivial task Workbench should search, in order:

1. current context capsule;
2. project memories;
3. matching project routines/assets;
4. matching global memories;
5. matching global routines/assets;
6. repository source;
7. autonomous exploration only when still useful.

This makes prior successful work cheaper than re-discovery.

## Initial implementation

The current implementation provides a local JSON-backed knowledge store with project/global memories, routines, context capsules, secret-like-content refusal and deterministic text/tag search. MCP exposes read/write tools for lead chats, and autonomous worker prompts automatically include a bounded selection of relevant memory and the latest project capsule. A future embedding index can improve retrieval without changing the durable data model.
