# Durable compact context and reusable knowledge

Workbench must not depend on one indefinitely growing conversation. Conversation history is a working buffer; durable project state belongs in Workbench.

## Goals

- A fresh chat can resume a project from a small context pack instead of replaying its full history.
- Project decisions and constraints survive conversation compaction.
- Useful lessons can be project-scoped or deliberately promoted to cross-project memory.
- Proven routines and code templates are stored once and retrieved for similar work instead of being rebuilt repeatedly.
- Autonomous workers receive relevant prior knowledge automatically without consuming the entire memory database.
- Secrets never become ordinary model-facing memory.

## Storage model

Workbench keeps the knowledge database in local protected application state, separate from the public source tree and separate from the ordinary task queue.

### Project identity

When a repository has an `origin` Git remote, Workbench normalises it into a portable project key. This lets a project retain one identity even when the local checkout path changes. Repositories without a usable remote get a machine-local fallback identity.

### Memory scopes

`project` memory belongs only to one project identity. Typical items are architecture decisions, constraints, project-specific lessons and successful task outcomes.

`global` memory is deliberately reusable across projects. Typical items are engineering patterns, review rules, deployment-independent techniques and other knowledge that should be recalled elsewhere.

Promotion to global scope is explicit. Workbench does not automatically turn project-specific material into cross-project knowledge.

### Checkpoints

A checkpoint is the compact state required to continue in a fresh conversation:

- current summary;
- durable decisions;
- open loops;
- likely next actions.

Workbench retains recent checkpoints, but context packs use the latest checkpoint rather than replaying old checkpoints.

### Routines and code

A routine is a named reusable procedure with:

- project or global scope;
- description and retrieval triggers;
- ordered steps;
- optional reusable code/template;
- language and tags.

Saving the same routine name in the same scope updates it instead of creating another copy. This is the first deduplication boundary for “we have already solved this kind of thing”.

## Retrieval and compaction

`get_context_pack` builds a bounded context object from:

1. the latest project checkpoint;
2. relevant project memory;
3. relevant global memory;
4. relevant project/global routines.

Retrieval uses inexpensive deterministic lexical ranking today. The storage model is intentionally independent of the ranking implementation so embeddings or a local semantic index can be added later without rewriting the durable data.

The rendered context has a hard character budget. It is therefore safe to use as the hand-off between conversations, models and workers without turning the full project history into prompt baggage.

## Worker behaviour

Tasks delegated through Workbench MCP use `DelegateWithKnowledge`. Workbench retrieves a bounded context pack for the requested intent and attaches it to the worker instructions as prior knowledge. Recorded decisions and constraints are treated as defaults unless the current request explicitly supersedes them.

Successful autonomous tasks are compacted into project-scoped outcome memories. Secret-looking output is skipped rather than retained.

## Model-facing tools

Read-only:

- `get_context_pack`
- `recall_memory`
- `find_routines`

Write:

- `remember`
- `save_checkpoint`
- `save_routine`

All model-facing writes reject probable secret material. Raw vault values remain outside the memory system.

## Conversation lifecycle

A lead AI should treat conversation length as disposable:

1. Start or resume with `get_context_pack`.
2. Work normally using repository eyes/hands and autonomous workers as needed.
3. Save distilled decisions/constraints when they become durable.
4. Save or update a routine when a procedure/code pattern has proven reusable.
5. At a meaningful milestone, or before context becomes unwieldy, write one new checkpoint.
6. A fresh conversation can then continue from the latest checkpoint plus relevant memory.

The desired result is not “infinite chat history”. It is **small active context backed by durable structured memory**.

## Next layers

The initial store is deliberately simple and local. Planned layers include:

- personal-plan Git-relay control envelopes for checkpoint/memory writes when direct custom-MCP actions are unavailable;
- automatic checkpoint suggestions based on task/conversation state rather than arbitrary token thresholds;
- semantic retrieval and memory consolidation;
- routine usage/provenance telemetry;
- reusable artefact/file references larger than inline code snippets;
- optional encrypted/synchronised private knowledge backends;
- retention, archive and explicit forgetting controls.
