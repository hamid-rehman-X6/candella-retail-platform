"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { listStores, type StoreLocation } from "@/lib/store-api";

export function useStoreSelection(tenantId: string) {
  const [stores, setStores] = useState<StoreLocation[]>([]);
  const [selection, setSelection] = useState<{
    tenantId: string;
    id: string;
  } | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const loadID = useRef(0);

  const refresh = useCallback(async () => {
    const requestID = ++loadID.current;
    setLoading(true);
    try {
      const data = await listStores(tenantId);
      if (requestID !== loadID.current) return;
      setStores(data.stores);
      const stored = sessionStorage.getItem(`candella.store.${tenantId}`);
      const selected =
        data.stores.find((store) => store.id === stored && store.accessible) ??
        data.stores.find((store) => store.isDefault && store.accessible) ??
        data.stores.find((store) => store.accessible);
      setSelection({ tenantId, id: selected?.id ?? "" });
      setError("");
    } catch (cause) {
      if (requestID !== loadID.current) return;
      setSelection({ tenantId, id: "" });
      setError(
        cause instanceof Error ? cause.message : "Could not load stores.",
      );
    } finally {
      if (requestID === loadID.current) setLoading(false);
    }
  }, [tenantId]);

  useEffect(() => {
    const requestRef = loadID;
    const timer = setTimeout(() => {
      void refresh();
    }, 0);
    return () => {
      clearTimeout(timer);
      requestRef.current++;
    };
  }, [refresh]);

  const storeId = selection?.tenantId === tenantId ? selection.id : "";
  const selectedStore = stores.find((store) => store.id === storeId) ?? null;
  function select(id: string) {
    if (!stores.some((store) => store.id === id && store.accessible)) return;
    setSelection({ tenantId, id });
    sessionStorage.setItem(`candella.store.${tenantId}`, id);
  }

  return { stores, storeId, selectedStore, select, loading, error, refresh };
}
