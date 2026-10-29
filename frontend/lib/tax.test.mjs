import { test } from "node:test";
import assert from "node:assert/strict";
import { formatRate, lineGross, lineTax, share } from "./tax.ts";

// The same cases as backend/internal/tax/calc_test.go, so the till and the
// server can never disagree about a line's tax.
test("lineTax matches the server's rounding", () => {
  const cases = [
    [150000, 0, true, 0],
    [0, 1800, false, 0],
    [100000, 1800, false, 18000],
    [118000, 1800, true, 18000],
    [45000, 1800, true, 6864],
    [33, 1500, false, 5],
    [10, 500, false, 1],
    [9, 500, false, 0],
    [2000, 10000, true, 1000],
    [89000, 1700, false, 15130],
    [33, 1700, false, 6],
  ];
  for (const [line, bps, inclusive, want] of cases) {
    assert.equal(lineTax(line, bps, inclusive), want, `${line} @ ${bps}`);
  }
});

test("lineGross adds only exclusive tax", () => {
  assert.equal(lineGross(100000, 18000, false), 118000);
  assert.equal(lineGross(118000, 18000, true), 118000);
});

test("share splits a line so the parts add up exactly", () => {
  for (const parts of [[1, 1, 1], [2, 1], [1, 2], [3]]) {
    let already = 0;
    let sum = 0;
    for (const take of parts) {
      sum += share(1000, 3, already, take);
      already += take;
    }
    assert.equal(sum, 1000);
  }
  assert.equal(share(1000, 3, 0, 1), 333);
  assert.equal(share(1000, 3, 2, 1), 334);
});

test("formatRate shows a percentage", () => {
  assert.equal(formatRate(1800), "18%");
  assert.equal(formatRate(1750), "17.5%");
  assert.equal(formatRate(0), "0%");
});
