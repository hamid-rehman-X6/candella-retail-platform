"use client";

import Link from "next/link";
import { useCallback, useEffect, useState, type FormEvent } from "react";
import { Archive, Pencil, Search, Users } from "lucide-react";
import { useApp, workspacePath } from "@/components/app/app-context";
import { formatDateTime } from "@/lib/datetime";
import { formatMoney } from "@/lib/money";
import {
  createCustomer,
  customerSales,
  listCustomers,
  updateCustomer,
  type Customer,
  type CustomerInput,
  type CustomerSale,
} from "@/lib/customers-api";
import { toast } from "@/lib/toast";

const field =
  "w-full rounded-xl border border-line bg-surface px-3 py-2.5 text-sm text-foreground outline-none focus:border-accent";
const blank: CustomerInput = { name: "", phone: "", email: "" };

export default function CustomersPage() {
  const { current } = useApp();
  return <CustomersWorkspace key={current.id} />;
}

function CustomersWorkspace() {
  const { current } = useApp();
  const manager = current.role === "owner" || current.role === "admin";
  const [search, setSearch] = useState("");
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [offset, setOffset] = useState(0);
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState<Customer | null>(null);
  const [history, setHistory] = useState<CustomerSale[]>([]);
  const [historyOffset, setHistoryOffset] = useState(0);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [form, setForm] = useState<CustomerInput>(blank);
  const [editing, setEditing] = useState<string | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const result = await listCustomers(current.id, search, offset);
      setCustomers(result.customers);
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not load customers.");
    } finally {
      setLoading(false);
    }
  }, [current.id, search, offset]);

  useEffect(() => {
    let cancelled = false;
    const timer = setTimeout(async () => {
      try {
        const result = await listCustomers(current.id, search, offset);
        if (!cancelled) {
          setCustomers(result.customers);
          setError("");
          setLoading(false);
        }
      } catch (e) {
        if (!cancelled) {
          setError(
            e instanceof Error ? e.message : "Could not load customers.",
          );
          setLoading(false);
        }
      }
    }, 200);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [current.id, search, offset]);

  useEffect(() => {
    if (!selected || !manager) return;
    let cancelled = false;
    const timer = setTimeout(() => {
      setHistoryLoading(true);
      void customerSales(current.id, selected.id, historyOffset)
        .then(({ sales }) => {
          if (!cancelled) setHistory(sales);
        })
        .catch((e) => {
          if (!cancelled)
            setError(e instanceof Error ? e.message : "Could not load sales.");
        })
        .finally(() => {
          if (!cancelled) setHistoryLoading(false);
        });
    }, 0);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [current.id, selected, historyOffset, manager]);

  function openForm(customer?: Customer) {
    setEditing(customer?.id ?? null);
    setForm(
      customer
        ? { name: customer.name, phone: customer.phone, email: customer.email }
        : blank,
    );
    setFormOpen(true);
  }
  async function save(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      const { customer } = editing
        ? await updateCustomer(current.id, editing, form)
        : await createCustomer(current.id, form);
      setSelected(customer);
      setHistory([]);
      setHistoryLoading(true);
      setFormOpen(false);
      toast.success(editing ? "Customer updated" : "Customer added");
      await refresh();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Could not save customer.");
    } finally {
      setBusy(false);
    }
  }
  async function changeStatus(customer: Customer) {
    setBusy(true);
    try {
      const next = customer.status === "active" ? "archived" : "active";
      const result = await updateCustomer(current.id, customer.id, {
        status: next,
      });
      setSelected(result.customer);
      toast.success(
        next === "active" ? "Customer restored" : "Customer archived",
      );
      await refresh();
    } catch (e) {
      toast.error(
        e instanceof Error ? e.message : "Could not update customer.",
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-6xl space-y-6 pb-12">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-3xl font-semibold">Customers</h1>
          <p className="mt-1 text-sm text-muted">
            Keep contact details and purchase history in your workspace.
          </p>
        </div>
        {manager && (
          <button
            type="button"
            onClick={() => openForm()}
            className="cursor-pointer rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-black hover:bg-accent-bright"
          >
            Add customer
          </button>
        )}
      </div>
      {error && (
        <p
          role="alert"
          className="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-danger-ink"
        >
          {error}
        </p>
      )}
      <label className="block max-w-xl text-sm font-medium">
        Search customers
        <span className="relative mt-2 block">
          <Search className="absolute top-3 left-3 h-4 w-4 text-muted" />
          <input
            className={`${field} pl-10`}
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setOffset(0);
              setLoading(true);
            }}
            placeholder="Name, phone, or email"
          />
        </span>
      </label>
      {formOpen && manager && (
        <form
          onSubmit={save}
          className="space-y-4 rounded-2xl border border-line bg-surface p-5"
        >
          <h2 className="text-lg font-semibold">
            {editing ? "Edit customer" : "New customer"}
          </h2>
          <div className="grid gap-4 sm:grid-cols-3">
            {(["name", "phone", "email"] as const).map((key) => (
              <label key={key} className="text-sm font-medium capitalize">
                {key}
                <input
                  className={`${field} mt-2`}
                  type={key === "email" ? "email" : "text"}
                  maxLength={key === "name" ? 160 : key === "phone" ? 40 : 254}
                  value={form[key]}
                  onChange={(e) => setForm({ ...form, [key]: e.target.value })}
                  required={key === "name"}
                />
              </label>
            ))}
          </div>
          <p className="text-xs text-muted">
            Add a phone or email. Active customers cannot share either contact
            detail.
          </p>
          <div className="flex gap-3">
            <button
              disabled={busy}
              className="cursor-pointer rounded-xl bg-accent px-4 py-2 text-sm font-semibold text-black disabled:opacity-50"
            >
              {busy ? "Saving…" : "Save customer"}
            </button>
            <button
              type="button"
              onClick={() => setFormOpen(false)}
              className="cursor-pointer rounded-xl border border-line px-4 py-2 text-sm"
            >
              Cancel
            </button>
          </div>
        </form>
      )}
      <div className="grid items-start gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(300px,400px)]">
        <section
          className="overflow-hidden rounded-2xl border border-line bg-surface"
          aria-label="Customer directory"
        >
          {loading ? (
            <p className="p-6 text-sm text-muted">Loading customers…</p>
          ) : customers.length === 0 ? (
            <p className="p-6 text-sm text-muted">
              No customers found. Walk-in sales are still available.
            </p>
          ) : (
            customers.map((customer) => (
              <button
                key={customer.id}
                type="button"
                onClick={() => {
                  setSelected(customer);
                  setHistory([]);
                  setHistoryLoading(true);
                  setHistoryOffset(0);
                }}
                className={`flex w-full cursor-pointer items-center justify-between gap-3 border-b border-line p-4 text-left last:border-0 hover:bg-foreground/5 ${selected?.id === customer.id ? "bg-accent/10" : ""}`}
              >
                <span className="min-w-0">
                  <span className="block truncate font-medium">
                    {customer.name}
                  </span>
                  {manager && (
                    <span className="mt-1 block truncate text-xs text-muted">
                      {customer.phone || customer.email || "No contact"}
                    </span>
                  )}
                </span>
                <span className="shrink-0 text-xs text-muted">
                  {customer.status === "archived" ? "Archived" : "View"}
                </span>
              </button>
            ))
          )}
          <div className="flex items-center justify-between border-t border-line p-3 text-sm">
            <button
              type="button"
              disabled={offset === 0}
              onClick={() => {
                setOffset(Math.max(0, offset - 50));
                setLoading(true);
              }}
              className="cursor-pointer disabled:opacity-40"
            >
              Previous
            </button>
            <span className="text-muted">Page {offset / 50 + 1}</span>
            <button
              type="button"
              disabled={customers.length < 50}
              onClick={() => {
                setOffset(offset + 50);
                setLoading(true);
              }}
              className="cursor-pointer disabled:opacity-40"
            >
              Next
            </button>
          </div>
        </section>
        {selected && (
          <section
            className="space-y-4 rounded-2xl border border-line bg-surface p-5"
            aria-label="Customer details"
          >
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div>
                <h2 className="text-lg font-semibold">{selected.name}</h2>
                <p className="text-xs text-muted">
                  {selected.status === "archived" ? "Archived" : "Active"}
                </p>
              </div>
              {manager && (
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={() => openForm(selected)}
                    className="cursor-pointer rounded-lg border border-line p-2"
                    aria-label="Edit customer"
                  >
                    <Pencil className="h-4 w-4" />
                  </button>
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => void changeStatus(selected)}
                    className="cursor-pointer rounded-lg border border-line px-3 py-2 text-xs disabled:opacity-40"
                  >
                    <Archive className="mr-1 inline h-4 w-4" />
                    {selected.status === "active" ? "Archive" : "Restore"}
                  </button>
                </div>
              )}
            </div>
            {manager ? (
              <>
                <p className="text-sm text-muted">
                  Phone: {selected.phone || "—"}
                  <br />
                  Email: {selected.email || "—"}
                </p>
                <h3 className="border-t border-line pt-4 font-semibold">
                  Sales history
                </h3>
                {historyLoading ? (
                  <p className="text-sm text-muted">Loading sales…</p>
                ) : history.length === 0 ? (
                  <p className="text-sm text-muted">
                    No sales linked to this customer yet.
                  </p>
                ) : (
                  history.map((sale) => (
                    <Link
                      key={sale.id}
                      href={workspacePath(current.slug, `/sales/${sale.id}`)}
                      className="flex justify-between gap-2 rounded-xl border border-line p-3 text-sm hover:border-accent/50"
                    >
                      <span>
                        <span className="block font-medium">
                          {sale.orderNumber}
                        </span>
                        <span className="text-xs text-muted">
                          {sale.storeName} ·{" "}
                          {formatDateTime(sale.createdAt, current.timezone)}
                        </span>
                      </span>
                      <span className="font-medium">
                        {formatMoney(sale.grandTotalMinor, sale.currencyCode)}
                      </span>
                    </Link>
                  ))
                )}
                <div className="flex justify-between text-sm">
                  <button
                    disabled={historyOffset === 0}
                    onClick={() =>
                      setHistoryOffset(Math.max(0, historyOffset - 50))
                    }
                    className="cursor-pointer disabled:opacity-40"
                  >
                    Previous
                  </button>
                  <button
                    disabled={history.length < 50}
                    onClick={() => setHistoryOffset(historyOffset + 50)}
                    className="cursor-pointer disabled:opacity-40"
                  >
                    Next
                  </button>
                </div>
              </>
            ) : (
              <p className="flex items-center gap-2 text-sm text-muted">
                <Users className="h-4 w-4" />
                Contact details and history are available to owners and admins.
              </p>
            )}
          </section>
        )}
      </div>
    </div>
  );
}
