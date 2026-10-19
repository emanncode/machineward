---
name: machineward-engineering
description: Use when planning, implementing, or reviewing work in Machineward. Tracks code work in Linear, compares viable maintained approaches when tradeoffs matter, and keeps implementation understandable to EmannCode.
---

# Machineward Engineering

Use this workflow for code and technical decisions in the Machineward repository. The goal is working software that EmannCode understands and can explain as portfolio work.

## Linear workflow

Use the EmannCode team statuses in order:

1. **Backlog**: State the user outcome, scope, constraints, and acceptance criteria. No implementation has started.
2. **In Progress**: Begin discovery and implementation. Before changing code, inspect repository instructions, Git status, relevant code and call sites, configuration, and existing project decisions. Preserve unrelated changes.
3. **In Review**: The change is ready for a behavior and code review. Record what changed, how it works, why the approach was chosen, checks actually run, limitations, and open findings. Walk EmannCode through the important logic.
4. **Done**: Acceptance criteria are met, review findings are resolved or explicitly accepted for follow-up, verification is accurately recorded, and the completed implementation's code review document is linked from the issue.

For code work, ensure a Linear issue is in Backlog with an outcome and acceptance criteria before substantive implementation. Move it to In Progress when work starts. Use the configured **In Review** status. Do not create a separate Working status or use Todo for planned Machineward work.

## Decide before coding

- Separate the desired product behavior from the user's technical diagnosis or suggested implementation.
- Establish the current behavior, expected behavior, relevant interfaces, constraints, and what evidence would show success.
- Challenge assumptions from both the user and the agent using code, configuration, documentation, or observable behavior.
- For routine work with one clear, maintained, non-deprecated approach, state the choice briefly and proceed.
- Compare approaches only when at least two maintained, non-deprecated options could reasonably satisfy the requirement. Compare material consequences such as correctness, security, complexity, usability, compatibility, performance, dependencies, and how well EmannCode can maintain and explain the result.
- Recommend one approach with reasons. Discuss consequential backend, frontend, data, security, or architecture choices before a large implementation. Do not manufacture alternatives or delay routine work.

## Implement in reviewable slices

- Keep changes tied to the agreed scope and acceptance criteria.
- Explain significant code as it is introduced. Avoid large unexplained implementations when the task is part of EmannCode's learning or portfolio work.
- Keep architecture proportional to the current V0. Revisit decisions when new evidence or requirements make the original tradeoff invalid.
- Update Linear when scope or material findings change. Record deferred product work as a Backlog issue.

## Review and verify

- Explain the data flow and important control flow in plain language so EmannCode can describe the implementation during a code review.
- Review edge cases, failure behavior, security boundaries, and relevant regressions.
- Plan proportionate verification. Run checks only when authorized by the user or higher-priority instructions, and report exact observed results. Never claim a build, test, runtime, or deployment succeeded unless it was verified.
- Keep the issue in In Review while consequential findings or user questions remain unresolved. Do not mark Done merely because files changed or code compiled.

## Linear code review documents

- For an implementation issue with code changes, create or update a standalone Linear code review document when the work reaches In Review. Attach it to the issue and link it from the project review record when one exists.
- Before moving that issue to Done, finalize and verify the document link. Treat this as part of the Done checkpoint, not optional cleanup after closure.
- Include the intended outcome and scope, files or components changed, a plain-language data/control flow walkthrough, approach and material tradeoffs, findings with resolved/open/accepted status, exact checks and observed results, limitations, commit references, and deployment/runtime status. Distinguish what was tested from what was not tested.
- Keep issue-specific reviews separate so a reader can find the implementation and reasoning for each completed slice. Keep the first development slice review as the broad historical review and add links to later slice reviews there.
- When documenting an issue that was already Done before this rule, create a retrospective review from available evidence. Label it as retrospective and do not invent checks or decisions that were not recorded.
- Do not create code review documents for backlog items, planning-only work, or issues with no meaningful code change unless the user asks for one.

## Communication

Use concise, direct language. Do not use the em dash character in user-facing Machineward updates or documents.
