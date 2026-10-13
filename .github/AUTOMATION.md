# Candella GitHub delivery flow

## What runs

1. A maintainer creates an issue with the feature form, or runs **Seed approved issues** after adding reviewed entries to `.github/roadmap.json`. Seeded issues are deduplicated by their stable `id` marker and start in `agent:triage`.
2. A maintainer assigns exactly one `priority:P0` to `priority:P3` label and adds `agent:ready`. This is the scope approval gate. `P0` runs first, then `P1`, `P2`, `P3`; ties use issue number. An `agent:blocked` issue is skipped. Only one `agent:working` issue may exist.
3. **Implement next issue** runs on the ready label, after a merged PR, hourly, or by manual dispatch. It creates `feature/<issue-number>-<title>`, asks Codex to implement the issue, blocks changes to automation files, and opens a PR that closes the issue when merged. A GitHub App token is used for the push and PR so normal PR workflows run.
4. **CI** runs frontend checks, backend checks, queue tests, and migrations against a disposable PostgreSQL service. **Agent review** checks the PR diff and fails on actionable findings.
5. With the required checks configured, `CANDELLA_AUTO_MERGE=true` enables auto merge only for PRs that do not touch migrations, auth, catalog, workspace, tenant database code, or auth UI. Those sensitive PRs receive `needs-human-review` and wait for a maintainer to merge them.
6. A failed implementation marks its issue `agent:blocked` and comments with the workflow run. Fix the problem, remove `agent:blocked`, and add `agent:ready` to retry. A failed PR stays open for repair; the queue waits until the associated issue closes or is manually moved out of `agent:working`.

## One-time GitHub setup

These settings live on GitHub and cannot be activated by merging workflow files alone.

1. Enable Issues and Actions for this repository. Run **Seed approved issues** once to create the labels, even when the roadmap is empty.
   Each roadmap entry needs a unique `id`, `title`, `priority` (`P0` to `P3`), `userNeed`, `acceptance` (string array), and `verification` (string array). Optional `rollout` explains dependencies or manual steps. Edit this file through a reviewed PR before seeding.
2. Create and install a GitHub App limited to this repository. Grant repository **Contents: read/write**, **Issues: read/write**, and **Pull requests: read/write**. Store its App ID in `CANDELLA_APP_ID` and private key in `CANDELLA_APP_PRIVATE_KEY` as Actions secrets. Store its bot login (for example `your-app[bot]`) in the `CANDELLA_APP_BOT_LOGIN` Actions variable.
3. Add an `OPENAI_API_KEY` Actions secret with access to Codex. The official `openai/codex-action` uses it for both implementation and review. Do not add keys to issues, PRs, workflow files, or `.env.example`.
4. Protect `main`: require a PR, require the exact checks **Frontend (lint, typecheck, build)**, **Backend (vet, build, test)**, **Automation (queue tests)**, and **Agent review**, and block direct pushes. Require conversations to be resolved. If there are other maintainers, require their review for sensitive changes as well.
5. Enable repository auto merge. Only after confirming the branch protection and required checks, set repository Actions variable `CANDELLA_AUTO_MERGE` to `true`. Until then, PRs wait for a manual merge.
6. Add focused tests and a rollout note for each business issue. Database migration commands against the owner's database are run by the owner, never by the agent. The CI database is disposable.

The App installation token is short lived and scoped to this repository. PRs created with the default `GITHUB_TOKEN` do not reliably start the follow-on PR workflows, which is why the App is needed. Issue labels use `GITHUB_TOKEN` because the same workflow immediately consumes the selected issue.

## Current Products/Inventory changes

The current local branch is `feature/products-inventory`. Create an issue with the feature form before opening its PR; include product creation, variants, stock receipt, count correction, tenant isolation, and the local migration rollout steps. Link the issue in the PR body. The branch contains migrations `0014` and `0015`, so it must wait for human review and the owner-run migration/smoke test described in `docs/features/01_products_inventory.md` (local documentation). It must not be auto merged simply because compile checks pass.

## References

- [GitHub issue forms](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/configuring-issue-templates-for-your-repository)
- [GitHub required checks and branch protection](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)
- [GitHub auto merge](https://docs.github.com/en/pull-requests/how-tos/merge-and-close-pull-requests/automatically-merging-a-pull-request)
- [Codex GitHub Action](https://developers.openai.com/codex/github-action)
