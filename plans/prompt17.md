Draft a README.md for the Conductor repository.

OBJECTIVE

Create a concise, human-oriented README that gives a developer or evaluator a quick understanding of what Conductor is, why it exists, how it is used, and where to start.

The README is NOT a second AGENTS.md.

AGENTS.md is the authoritative document for architecture, invariants, boundaries, implementation rules, lifecycle semantics, security constraints, growth gates, and detailed engineering guidance. Do not duplicate that material in README.md.

SOURCE OF TRUTH

First inspect the actual repository and read the existing AGENTS.md.

Use the repository contents and AGENTS.md as the source of truth. Do not invent capabilities, endpoints, architectural responsibilities, or future plans that are not supported by the repository.

README.md should summarize the project for humans; AGENTS.md should remain the detailed engineering authority.

README CONTENT

Include only what is useful for someone encountering Conductor for the first time:

1. Project title
   - Conductor

2. One-paragraph description
   - Explain Conductor in plain language.
   - Emphasize that it is a lightweight coordination system for autonomous/AI-driven work.
   - Explain the practical problem it solves without turning this into an architecture essay.

3. Core model
   Give a very concise explanation of the four roles:
   - Agent — agency
   - Conductor — coordination
   - Solvent — authority
   - External executor — effect

   Present this as an easy-to-understand mental model, not as a duplicate of the detailed architecture in AGENTS.md.

4. What Conductor provides
   Summarize the actual user-facing capabilities currently implemented, such as:
   - projects
   - tasks
   - task dependencies
   - task assignment/claiming
   - lifecycle management
   - activity history
   - HTTP API
   - MCP interface
   - minimal web UI
   - read-only governance observation

   Keep this high level. Do not reproduce endpoint-by-endpoint API documentation from AGENTS.md.

5. Typical workflow
   Show a very small example of how Conductor is used by an agent, for example:

   discover work
       ↓
   claim task
       ↓
   perform work
       ↓
   record activity / submit
       ↓
   review / accept

   Mention dependencies where useful.

6. What Conductor is NOT
   Keep this extremely concise.
   Explain that Conductor is not:
   - an AI reasoning engine
   - a domain/scientific knowledge system
   - an authorization engine
   - an execution system

   Do not reproduce the full boundary/invariant discussion from AGENTS.md.

7. Quick start
   Inspect the repository for the actual current commands and provide the minimum practical instructions needed to:
   - build
   - test
   - run Conductor

   Use the actual commands supported by the repository.
   Do not invent environment variables or startup procedures.
   Keep this section short.

8. Repository orientation
   Give a short explanation of the important top-level directories/files only when this genuinely helps a new human understand the repository.

9. Documentation
   Point readers to AGENTS.md for:
   - architecture
   - invariants
   - implementation rules
   - API/MCP details
   - security and lifecycle constraints
   - contribution/development boundaries

   Make it clear that AGENTS.md is the engineering source of truth.

10. License
   At the very bottom of README.md include exactly:

   ## License

   MIT

STYLE

- Concise and readable.
- Written for humans, not coding agents.
- Prefer explanation over exhaustive reference material.
- Avoid architectural jargon unless necessary.
- Do not restate AGENTS.md.
- Do not include speculative roadmap material.
- Do not include BM-IST-specific concepts; Conductor's README must remain domain-agnostic.
- Do not add badges, screenshots, diagrams, or decorative sections unless they already exist in the repository and materially improve the quick overview.
- Do not claim production readiness unless the repository explicitly establishes that.
- Do not change any source code.

DELIVERABLE

Produce only a proposed README.md draft and briefly explain the structure chosen.

Before writing it, inspect the repository and AGENTS.md so the README reflects the actual current implementation rather than assumptions.