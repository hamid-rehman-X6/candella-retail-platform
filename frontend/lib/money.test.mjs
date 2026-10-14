import { test } from "node:test";
import assert from "node:assert/strict";
import { formatMoney, minorDigits, parseMoney } from "./money.ts";

// Intl renders PKR with a no-break space between "Rs" and the amount.
const plain = (s) => s.replace(/\s/g, " ");

test("PKR uses two minor digits, matching the backend CSV importer", () => {
  assert.equal(minorDigits("PKR"), 2);
  assert.equal(parseMoney("150", "PKR"), 15000);
  assert.equal(parseMoney("149.50", "PKR"), 14950);
});

test("PKR formats as rupees, dropping .00 only for whole amounts", () => {
  assert.equal(plain(formatMoney(125000, "PKR")), "Rs 1,250");
  assert.equal(plain(formatMoney(6375, "PKR")), "Rs 63.75");
  assert.equal(plain(formatMoney(1234567800, "PKR")), "Rs 12,345,678");
});

test("a typed price round-trips to the same display", () => {
  const minor = parseMoney("1250", "PKR");
  assert.equal(plain(formatMoney(minor, "PKR")), "Rs 1,250");
});

test("rejects amounts with more decimals than the currency allows", () => {
  assert.equal(parseMoney("12.345", "PKR"), null);
  assert.equal(parseMoney("1,250", "PKR"), null);
  assert.equal(parseMoney("-5", "PKR"), null);
});

test("digit table matches the backend for non-2-digit currencies", () => {
  assert.equal(minorDigits("JPY"), 0);
  assert.equal(minorDigits("KWD"), 3);
  assert.equal(parseMoney("500", "JPY"), 500);
});
