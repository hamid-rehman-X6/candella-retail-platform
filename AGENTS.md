# Candella agent instructions

- Build for a retailer using a multi-tenant POS. Prefer clear operator flows and error messages.
- Read `README.md` and the issue before editing. Keep one issue per branch and PR.
- Preserve tenant isolation, authorization checks, append-only stock movements, and integer money amounts.
- Add meaningful tests for changed business behavior. Run Go tests and frontend checks.
- Never execute migrations against a developer or production database. Document the required command in the PR.
- Do not edit `.github/workflows/`, `.github/scripts/`, or this file for a product issue.
- Do not commit secrets, local `.env` files, or generated artifacts.
- If the issue is unclear or unsafe, explain the blocker in the final message instead of guessing.
