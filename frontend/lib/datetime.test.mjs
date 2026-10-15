import { test } from "node:test";
import assert from "node:assert/strict";
import { DEFAULT_TIMEZONE, formatDate, formatDateTime } from "./datetime.ts";

// Intl may separate the time and "am"/"pm" with a narrow no-break space.
const plain = (s) => s.replace(/\s/g, " ");

test("defaults to Pakistan time", () => {
  assert.equal(DEFAULT_TIMEZONE, "Asia/Karachi");
  assert.equal(
    plain(formatDateTime("2026-10-15T07:30:00Z")),
    "15-Oct-2026, 12:30 pm",
  );
});

test("a late-evening UTC sale lands on the next Pakistan day", () => {
  // 21:05 UTC on the 14th is 02:05 PKT on the 15th: the bug this fixes.
  assert.equal(
    plain(formatDateTime("2026-10-14T21:05:00Z", "Asia/Karachi")),
    "15-Oct-2026, 2:05 am",
  );
  assert.equal(
    formatDate("2026-10-14T21:05:00Z", "Asia/Karachi"),
    "15-Oct-2026",
  );
});

test("honours a workspace set to another zone", () => {
  assert.equal(
    plain(formatDateTime("2026-10-14T21:05:00Z", "UTC")),
    "14-Oct-2026, 9:05 pm",
  );
});
