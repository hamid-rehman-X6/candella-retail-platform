# Candella GitHub delivery flow

## What runs

1. A maintainer creates an issue with the feature form, or runs **Seed approved issues** after adding reviewed entries to `.github/roadmap.json`. Seeded issues are deduplicated by their stable `id` marker and start in `agent:triage`.
2. A maintainer assigns exactly one `priority:P0` to `priority:P3` label and adds `agent:ready`. This is the scope approval gate. `P0` runs first, then `P1`, `P2`, `P3`; ties use issue number. An `agent:blocked` issue is skipped. Only one `agent:working` issue may exist.
3. **Implement next issue** runs on the ready label, after a merged PR, hourly, or by manual dispatch. When `OPENAI_API_KEY` is configured, it creates `feature/<issue-number>-<title>`, asks Codex to implement the issue, blocks changes to automation files, and opens a PR that closes the issue when merged. A GitHub App token is used for the push and PR so normal PR workflows run. Without the key, it skips the issue and leaves its labels unchanged.
4. **CI** runs frontend checks, backend checks, queue tests, and migrations against a disposable PostgreSQL service. When `OPENAI_API_KEY` is configured, **Agent review** checks the PR diff and fails on actionable findings. Without the key, the check succeeds with a visible skipped notice; the owner reviews the PR manually.
5. Every PR waits for the repository owner to review and merge it manually after required checks pass. PRs touching migrations, auth, catalog, workspace, tenant database code, or auth UI receive `needs-human-review` to highlight the extra review and rollout work.
6. A failed implementation marks its issue `agent:blocked` and comments with the workflow run. Fix the problem, remove `agent:blocked`, and add `agent:ready` to retry. A failed PR stays open for repair; the queue waits until the associated issue closes or is manually moved out of `agent:working`.

## One-time GitHub setup

These settings live on GitHub and cannot be activated by merging workflow files alone.

1. Enable Issues and Actions for this repository. Run **Seed approved issues** once to create the labels, even when the roadmap is empty.
   Each roadmap entry needs a unique `id`, `title`, `priority` (`P0` to `P3`), `userNeed`, `acceptance` (string array), and `verification` (string array). Optional `rollout` explains dependencies or manual steps. Edit this file through a reviewed PR before seeding.
2. Create and install a GitHub App limited to this repository. Grant repository **Contents: read/write**, **Issues: read/write**, and **Pull requests: read/write**. Store its App ID in `CANDELLA_APP_ID` and private key in `CANDELLA_APP_PRIVATE_KEY` as Actions secrets. Store its bot login (for example `your-app[bot]`) in the `CANDELLA_APP_BOT_LOGIN` Actions variable.
3. Optional: add an `OPENAI_API_KEY` Actions secret with access to Codex to enable automatic issue implementation and PR review. Without it, automatic implementation is skipped and the owner handles code and review manually. Do not add keys to issues, PRs, workflow files, or `.env.example`.
4. Protect `main`: require a PR, require the exact checks **Frontend (lint, typecheck, build)**, **Backend (vet, build, test)**, **Automation (queue tests)**, and **Agent review**, and block direct pushes. Without the key, **Agent review** reports that it skipped and succeeds; this does not mean AI approved the PR. Require conversations to be resolved. If there are other maintainers, require their review for sensitive changes as well.
5. Keep repository auto merge disabled. The owner merges each approved PR manually after the required checks pass.
6. Add focused tests and a rollout note for each business issue. Database migration commands against the owner's database are run by the owner, never by the agent. The CI database is disposable.

The App installation token is short lived and scoped to this repository. PRs created with the default `GITHUB_TOKEN` do not reliably start the follow-on PR workflows, which is why the App is needed. Issue labels use `GITHUB_TOKEN` because the same workflow immediately consumes the selected issue.

## Current Products/Inventory changes

The current branch is `feature/products-inventory`, tracked by issue #2 and PR #4. The branch contains migrations `0014` and `0015`, so it waits for the owner-run migration and browser smoke test described in `docs/features/01_products_inventory.md` (local documentation) before the owner merges it.

## References

- [GitHub issue forms](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/configuring-issue-templates-for-your-repository)
- [GitHub required checks and branch protection](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)
- [Codex GitHub Action](https://developers.openai.com/codex/github-action)
