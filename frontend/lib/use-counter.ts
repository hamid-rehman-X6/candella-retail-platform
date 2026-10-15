"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  listRegisters,
  rememberCounter,
  rememberedCounter,
  type Register,
} from "@/lib/register-api";

/** The counters of the selected store, and the one this device sells from. */
export function useCounter(tenantId: string, storeId: string) {
  const [registers, setRegisters] = useState<Register[]>([]);
  const [selection, setSelection] = useState<{ key: string; id: string }>({
    key: "",
    id: "",
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const loadID = useRef(0);
  const key = `${tenantId}.${storeId}`;

  const refresh = useCallback(async () => {
    if (!storeId) return;
    const requestID = ++loadID.current;
    try {
      const data = await listRegisters(tenantId, storeId);
      if (requestID !== loadID.current) return;
      setRegisters(data.registers);
      const remembered = rememberedCounter(tenantId, storeId);
      const selected =
        data.registers.find((register) => register.id === remembered) ??
        data.registers[0];
      setSelection({ key: `${tenantId}.${storeId}`, id: selected?.id ?? "" });
      setError("");
    } catch (cause) {
      if (requestID !== loadID.current) return;
      setRegisters([]);
      setError(
        cause instanceof Error ? cause.message : "Could not load counters.",
      );
    } finally {
      if (requestID === loadID.current) setLoading(false);
    }
  }, [tenantId, storeId]);

  useEffect(() => {
    const requestRef = loadID;
    const timer = setTimeout(() => {
      setLoading(true);
      void refresh();
    }, 0);
    return () => {
      clearTimeout(timer);
      requestRef.current++;
    };
  }, [refresh]);

  // A selection made for another store never leaks into this one.
  const registerId = selection.key === key ? selection.id : "";
  const register = registers.find((item) => item.id === registerId) ?? null;

  function select(id: string) {
    if (!registers.some((item) => item.id === id)) return;
    setSelection({ key, id });
    rememberCounter(tenantId, storeId, id);
  }

  return { registers, registerId, register, select, loading, error, refresh };
}
