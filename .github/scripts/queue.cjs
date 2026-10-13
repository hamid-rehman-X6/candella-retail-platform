const PRIORITY = ["priority:P0", "priority:P1", "priority:P2", "priority:P3"];

function branchFor(issue) {
  const slug = issue.title
    .toLowerCase()
    .normalize("NFKD")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "")
    .slice(0, 45)
    .replace(/-$/g, "");
  return `feature/${issue.number}-${slug || "issue"}`;
}

function priorityOf(issue) {
  const labels = new Set(issue.labels.map((label) => label.name));
  const index = PRIORITY.findIndex((priority) => labels.has(priority));
  return index === -1 ? PRIORITY.length : index;
}

function nextIssue(issues) {
  if (
    issues.some(
      (issue) =>
        !issue.pull_request &&
        issue.labels.some((label) => label.name === "agent:working"),
    )
  ) {
    return null;
  }
  return (
    issues
      .filter(
        (issue) =>
          !issue.pull_request &&
          issue.labels.some((label) => label.name === "agent:ready") &&
          !issue.labels.some((label) => label.name === "agent:blocked") &&
          priorityOf(issue) < PRIORITY.length,
      )
      .sort(
        (a, b) => priorityOf(a) - priorityOf(b) || a.number - b.number,
      )[0] || null
  );
}

module.exports = { PRIORITY, branchFor, nextIssue };
