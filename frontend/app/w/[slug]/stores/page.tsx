"use client";

import { useCallback, useEffect, useState, type FormEvent } from "react";
import { useApp } from "@/components/app/app-context";
import { listStaff, type StaffMember } from "@/lib/staff-api";
import {
  createStore,
  listStoreAccess,
  listStores,
  renameStore,
  setDefaultStore,
  setStoreAccess,
  type StoreLocation,
} from "@/lib/store-api";

const field =
  "w-full rounded-xl border border-line bg-surface px-3 py-2.5 text-sm outline-none focus:border-accent";
const button =
  "rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-black disabled:opacity-50";

export default function StoresPage() {
  const { current } = useApp();
  const editable = current.role === "owner" || current.role === "admin";
  const [stores, setStores] = useState<StoreLocation[]>([]);
  const [members, setMembers] = useState<StaffMember[]>([]);
  const [name, setName] = useState("");
  const [selected, setSelected] = useState<string | null>(null);
  const [access, setAccess] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const refresh = useCallback(async () => {
    const [storeResult, staffResult] = await Promise.all([
      listStores(current.id),
      listStaff(current.id),
    ]);
    setStores(storeResult.stores);
    setMembers(
      staffResult.members.filter(
        (member) => member.status === "active" && member.role === "member",
      ),
    );
  }, [current.id]);

  useEffect(() => {
    if (!editable) return;
    const timer = setTimeout(() => {
      void refresh().catch((cause) =>
        setError(
          cause instanceof Error ? cause.message : "Could not load stores.",
        ),
      );
    }, 0);
    return () => clearTimeout(timer);
  }, [editable, refresh]);

  useEffect(() => {
    if (!selected) return;
    let active = true;
    void listStoreAccess(current.id, selected)
      .then(({ userIds }) => {
        if (active) setAccess(userIds);
      })
      .catch((cause) => {
        if (active)
          setError(
            cause instanceof Error
              ? cause.message
              : "Could not load store access.",
          );
      });
    return () => {
      active = false;
    };
  }, [current.id, selected]);

  async function run(action: () => Promise<unknown>, success: string) {
    setBusy(true);
    setError("");
    setMessage("");
    try {
      await action();
      await refresh();
      setMessage(success);
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Could not save. Try again.",
      );
    } finally {
      setBusy(false);
    }
  }

  async function add(event: FormEvent) {
    event.preventDefault();
    const value = name.trim();
    if (!value) return;
    await run(() => createStore(current.id, value), "Store created.");
    setName("");
  }

  async function rename(location: StoreLocation) {
    const next = window.prompt("Store name", location.name)?.trim();
    if (!next || next === location.name) return;
    await run(
      () => renameStore(current.id, location.id, next),
      "Store renamed.",
    );
  }

  async function changeAccess(userId: string, allowed: boolean) {
    if (!selected) return;
    await run(
      () => setStoreAccess(current.id, selected, userId, allowed),
      "Store access updated.",
    );
    const { userIds } = await listStoreAccess(current.id, selected);
    setAccess(userIds);
  }

  if (!editable)
    return (
      <p role="alert" className="text-sm text-muted">
        Only owners and admins can manage stores.
      </p>
    );
  const selectedStore = stores.find((store) => store.id === selected);

  return (
    <div className="mx-auto max-w-4xl space-y-6 pb-10">
      <div>
        <h1 className="text-2xl font-semibold">Stores</h1>
        <p className="mt-1 text-sm text-muted">
          Each location has its own stock. The default store remains available
          to every cashier.
        </p>
      </div>
      {error && (
        <p
          role="alert"
          className="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-300"
        >
          {error}
        </p>
      )}
      {message && (
        <p role="status" className="text-sm text-emerald-300">
          {message}
        </p>
      )}
      <form
        onSubmit={(event) => void add(event)}
        className="flex flex-wrap gap-2 rounded-2xl border border-line bg-surface p-5"
      >
        <label className="min-w-56 flex-1 text-sm font-medium">
          New store name
          <input
            className={`${field} mt-2`}
            value={name}
            maxLength={100}
            onChange={(event) => setName(event.target.value)}
            placeholder="e.g. Mall branch"
          />
        </label>
        <button
          type="submit"
          disabled={busy || !name.trim()}
          className={`${button} self-end`}
        >
          Add store
        </button>
      </form>
      <div className="space-y-3">
        {stores.map((store) => (
          <div
            key={store.id}
            className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-line bg-surface p-5"
          >
            <div>
              <p className="font-medium">
                {store.name}{" "}
                {store.isDefault && (
                  <span className="ml-2 rounded-full bg-accent/15 px-2 py-1 text-xs text-accent">
                    Default
                  </span>
                )}
              </p>
              <p className="mt-1 font-mono text-xs text-muted">{store.id}</p>
            </div>
            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                disabled={busy}
                onClick={() => void rename(store)}
                className="rounded-xl border border-line px-3 py-2 text-sm"
              >
                Rename
              </button>
              {!store.isDefault && (
                <button
                  type="button"
                  disabled={busy}
                  onClick={() =>
                    void run(
                      () => setDefaultStore(current.id, store.id),
                      `${store.name} is now the default store.`,
                    )
                  }
                  className="rounded-xl border border-line px-3 py-2 text-sm"
                >
                  Make default
                </button>
              )}
              {!store.isDefault && (
                <button
                  type="button"
                  onClick={() => setSelected(store.id)}
                  className="rounded-xl border border-line px-3 py-2 text-sm"
                >
                  Staff access
                </button>
              )}
            </div>
          </div>
        ))}
      </div>
      {selectedStore && !selectedStore.isDefault && (
        <section className="rounded-2xl border border-line bg-surface p-5">
          <div className="flex justify-between gap-4">
            <div>
              <h2 className="text-lg font-semibold">
                Staff access: {selectedStore.name}
              </h2>
              <p className="mt-1 text-sm text-muted">
                Owners and admins can use every store. Grant cashiers access to
                this location.
              </p>
            </div>
            <button
              type="button"
              onClick={() => setSelected(null)}
              className="text-sm text-accent"
            >
              Close
            </button>
          </div>
          <div className="mt-4 space-y-2">
            {members.length === 0 && (
              <p className="text-sm text-muted">No active cashiers yet.</p>
            )}
            {members.map((member) => (
              <label
                key={member.userId}
                className="flex items-center justify-between gap-4 rounded-xl border border-line p-3 text-sm"
              >
                <span>
                  {member.fullName}{" "}
                  <span className="text-muted">{member.email}</span>
                </span>
                <input
                  type="checkbox"
                  disabled={busy}
                  checked={access.includes(member.userId)}
                  onChange={(event) =>
                    void changeAccess(member.userId, event.target.checked)
                  }
                  aria-label={`Allow ${member.fullName} at ${selectedStore.name}`}
                />
              </label>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}
