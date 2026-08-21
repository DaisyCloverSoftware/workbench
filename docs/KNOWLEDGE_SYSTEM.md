# Workbench Knowledge System

Workbench should become more useful over time rather than restarting from zero whenever a chat context fills up or a new project begins.

The knowledge system is a **continuity/advisory system**, not a competing project specification. `docs/GOVERNANCE.md` defines the source-of-truth hierarchy.

The knowledge system has four layers:

1. **Conversation compaction** — distil a long working session into a bounded context capsule containing current objective, verified state, decisions/constraints to re-check against canonical docs, open threads and durable references.
2. **Project memory** — advisory facts/history for one repository: architecture history, conventions, commands, known traps, accepted trade-offs and project-specific procedures.
3. **Global memory** — reusable advisory knowledge across projects: operator preferences, environment-independent engineering patterns, tool behaviour and durable policies.
4. **Routine/library memory** — reusable recipes and implementation assets: tested command sequences, patches/templates, migration procedures, code snippets and small components that should be retrieved before rebuilding the same thing.

## Authority rule

Canonical repository documentation/decisions and verified current repository/runtime state outrank context capsules and memory.

A memory/capsule that conflicts with current canonical documentation is **stale evidence**, not authority. It must not silently override the project record or cause a superseded/rejected behaviour to return.

If a chat introduces a material new decision, that decision does not become durable project authority merely because it was saved to memory. The relevant canonical repository documentation must also be updated under `docs/GOVERNANCE.md`.

## Design rules

- Memory is structured, scoped and attributable; it is not one giant transcript summary.
- Every durable item has a scope, kind, source/provenance, timestamps and optional tags.
- Project memory never silently becomes global memory.
- Secret-like material is refused from model-readable memory and belongs in the vault instead.
- Compaction keeps a short active context capsule plus references to durable memories rather than copying the whole history forward.
- Retrieval is relevance-first and budgeted.
- Routines should be versionable and testable. A successful routine can be promoted from project scope to global scope explicitly.
- Generated code should be searched for reusable prior implementations before a new version is created.
- Autonomous workers receive only a bounded advisory context/memory slice.
- Canonical requirements, decision records, do-not-reintroduce rules and current source/runtime state MUST be checked before advisory memory can guide material implementation.
- Stale context/memory should be corrected or retired rather than repeatedly injected after its underlying decision has been superseded.

## Context capsule

A context capsule is a small continuity object used when a conversation is getting long. It should contain:

- current outcome;
- current repository/project identity;
- exact verified repository/runtime references where relevant;
- verified completed work using precise evidence states;
- active task IDs / branches / artifact references;
- unresolved questions;
- relevant memory/routine IDs;
- a bounded next-action summary;
- pointers to canonical documents that govern the next work.

A capsule may summarise a current decision but MUST point back to the canonical repository record rather than becoming the only place that decision exists.

## Retrieval / bootstrap order

Before implementing a non-trivial task:

1. read the repository's canonical governance/requirements/decision/architecture contracts relevant to the task;
2. verify current repository/runtime state needed for the task;
3. read the latest context capsule for continuity, treating it as advisory;
4. retrieve relevant project memories/routines/assets;
5. retrieve relevant global memories/routines/assets;
6. inspect implementation/tests/history as needed;
7. use autonomous exploration only when still useful.

When any advisory item conflicts with steps 1–2, canonical/current evidence wins and the stale advisory item should be corrected/retired.

## Initial implementation

The current implementation provides a local JSON-backed knowledge store with project/global memories, routines, context capsules, secret-like-content refusal and deterministic text/tag search. MCP exposes read/write tools for lead chats, and autonomous worker prompts can include a bounded selection of relevant memory and the latest project capsule.

The implementation must evolve to enforce the authority rule above: memory retrieval is useful context, never permission to override canonical repository decisions. A future embedding index can improve retrieval without changing that governance boundary.
