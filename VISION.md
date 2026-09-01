# Workbench Vision

## The developer should specify intent, not operate the AI switchboard

AI development tools are getting more capable while the workflow around them is getting more fragmented. A developer can have excellent reasoning in a normal chat, an autonomous coding agent with a limited weekly allowance, another vendor's subscription, a free local model, a cluster-side agent harness, a code review model and a collection of terminals.

The absurd part is that the human often becomes the integration layer.

Workbench exists to remove that job.

## The north-star interaction

The north-star command is deliberately boring:

> **Implement the next Workbench task.**

The user should be able to walk away. Workbench should continue until the work is complete or it encounters a decision that genuinely requires human authority. A notification then means exactly one thing: **log on; I need you**.

It must not mean “come watch me think”.

## Intelligence and agency are different resources

A central Workbench idea is that the strongest reasoning available to a developer is not always tied to the cheapest execution path.

Ordinary chat may be capable of understanding a bug and producing an exact patch. A local Workbench runner can apply that patch and run tests. There is no reason to consume scarce autonomous-agent allowance just to obtain file-system access.

So Workbench separates:

- **Brains:** planning, reasoning, review, code generation and diagnosis.
- **Hands:** file edits, safe commands, tests, diffs and execution.
- **Autonomy:** independent exploration and multi-step action only where the governing policy and current authority explicitly permit it.

For ChatGPT-driven engineering, ChatGPT remains responsible for the engineering loop and Workbench direct controls provide the normal machine-side hands. OpenClaw is not an automatic escalation route.

## A neutral AI IDE

Workbench should not belong to one model vendor.

Providers advertise capabilities and economic characteristics. A route can be local, included in a subscription, limited by a weekly/rolling agentic allowance, or metered by an API. Workbench chooses among routes that are already eligible under current authority rather than treating availability as permission.

It should be normal for one job to involve several systems when their separate policies permit it:

- a local model classifies logs;
- ChatGPT plans the change;
- Workbench applies an exact patch and runs tests;
- a separately authorised worker may perform an eligible non-OpenClaw task;
- a different model reviews authentication-sensitive code where policy permits;
- OpenClaw performs a machine operation only when the owner explicitly requested OpenClaw by name for that operation;
- the human hears nothing unless a real boundary is reached.

OpenClaw availability, an allowlist miss, missing direct capability, CI/deployment failure or operational difficulty is never authorization to select OpenClaw.

## User-owned context

Projects, notes, decisions, tasks and secrets belong to the user's workspace, not to whichever model happened to be active when the information was captured.

Workbench should be the durable context layer. AI providers receive only the context required for a task.

An idea can be parked without derailing active work. A secret can be stored without becoming prompt text. A task can survive a chat session and still have an auditable history.

## Open source by design

The useful public artifact is not one person's private productivity setup. It is the protocol, router, execution model and adapter ecosystem.

Workbench should make it straightforward for somebody to contribute:

- a new provider adapter;
- a new agent harness;
- a new notification transport;
- a policy module;
- a local runner;
- a test/preview surface.

The project succeeds if other developers can bring their own AI subscriptions and infrastructure without having to accept one company's vertically integrated stack.

## What we will not do

Workbench should not rely on brittle automation of consumer web sessions to pretend that every chat product has an API. It should use supported provider CLIs, APIs, MCP/tools and user-authorised harnesses.

Workbench should not silently ship secrets to models, silently deploy to production, turn every trivial job into an expensive agent run, or infer OpenClaw authorization from availability or failure of another execution path.

And it should never make the human babysit a progress spinner merely because the software cannot decide what to do next.
