const assert = require("node:assert/strict");
const test = require("node:test");
const { branchFor, nextIssue } = require("./queue.cjs");

const issue = (number, priority, extra = []) => ({
  number,
  title: `Issue ${number}`,
  labels: ["agent:ready", priority, ...extra].map((name) => ({ name })),
});

test("chooses the highest ready priority and oldest issue", () => {
  assert.equal(
    nextIssue([
      issue(1, "priority:P2"),
      issue(5, "priority:P0"),
      issue(3, "priority:P0"),
    ]).number,
    3,
  );
});

test("does not dispatch while another issue is working", () => {
  const working = issue(9, "priority:P2", ["agent:working"]);
  assert.equal(nextIssue([working, issue(2, "priority:P0")]), null);
});

test("skips blocked, unlabeled and pull request entries", () => {
  const pr = { ...issue(1, "priority:P0"), pull_request: {} };
  assert.equal(
    nextIssue([
      pr,
      issue(2, "priority:P0", ["agent:blocked"]),
      issue(3, "other"),
      issue(4, "priority:P1"),
    ]).number,
    4,
  );
});

test("makes a predictable safe feature branch", () => {
  assert.equal(
    branchFor({ number: 42, title: "POS: Checkout & Stock!" }),
    "feature/42-pos-checkout-stock",
  );
});
