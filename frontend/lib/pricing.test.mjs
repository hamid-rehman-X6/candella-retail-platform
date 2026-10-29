import { test } from "node:test";
import assert from "node:assert/strict";
import { PricingError, price } from "./pricing.ts";

// The same cases as backend/internal/pricing/pricing_test.go.
const plain = (lineMinor, quantity = 1) => ({
  lineMinor,
  quantity,
  taxRateBps: null,
  taxInclusive: null,
});
const taxed = (lineMinor, bps, inclusive, quantity = 1) => ({
  lineMinor,
  quantity,
  taxRateBps: bps,
  taxInclusive: inclusive,
});

test("no discount matches plain tax", () => {
  const r = price([
    taxed(45000, 1800, true),
    taxed(89000, 1700, false),
    plain(9000),
  ]);
  assert.equal(r.subtotalMinor, 143000);
  assert.equal(r.taxMinor, 21994);
  assert.equal(r.totalMinor, 158130);
});

test("an order percentage comes off before tax", () => {
  const r = price([plain(100000), taxed(100000, 1700, false)], {
    percentBps: 1000,
  });
  assert.equal(r.orderDiscountMinor, 20000);
  assert.deepEqual(
    r.lines.map((l) => l.discountMinor),
    [10000, 10000],
  );
  assert.equal(r.taxMinor, 15300);
  assert.equal(r.totalMinor, 195300);
});

test("an order discount spreads to the paisa", () => {
  const r = price([plain(1000), plain(1000), plain(1000)], {
    amountMinor: 100,
  });
  assert.equal(
    r.lines.reduce((s, l) => s + l.discountMinor, 0),
    100,
  );
  assert.equal(r.totalMinor, 2900);
});

test("item discounts are per unit or a percentage", () => {
  const r = price([
    { ...plain(30000, 3), item: { amountMinor: 2000 } },
    { ...plain(50000), item: { percentBps: 2500 } },
    plain(9000),
  ]);
  assert.deepEqual(
    r.lines.map((l) => l.discountMinor),
    [6000, 12500, 0],
  );
  assert.equal(r.totalMinor, 70500);
});

test("inclusive tax follows the discounted price", () => {
  const r = price([taxed(118000, 1800, true)], { percentBps: 5000 });
  assert.equal(r.totalMinor, 59000);
  assert.equal(r.taxMinor, 9000);
  assert.equal(r.includedTaxMinor, 9000);
});

test("refusals match the server", () => {
  assert.throws(
    () => price([plain(1000)], { amountMinor: 1001 }),
    PricingError,
  );
  assert.throws(
    () => price([{ ...plain(2000, 2), item: { amountMinor: 1001 } }]),
    PricingError,
  );
  assert.throws(
    () =>
      price([{ ...plain(1000), item: { percentBps: 100 } }], {
        percentBps: 100,
      }),
    (e) => e instanceof PricingError && e.code === "discount_combination",
  );
  assert.equal(
    price([taxed(1000, 1700, false)], { amountMinor: 1000 }).totalMinor,
    0,
  );
});
