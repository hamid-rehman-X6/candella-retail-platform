import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { getPlatformAdmin } from "@/lib/platform-api";
import { PlatformShell } from "@/components/platform/shell";

export const metadata: Metadata = {
  title: "Platform",
  robots: { index: false, follow: false },
};

/**
 * Guard for the whole platform console.
 *
 * Anyone who isn't active staff gets a 404, not a redirect and not a 403 — the
 * console shouldn't confirm its own existence to somebody probing for it. The
 * backend applies the same rule, so this is defence in depth rather than the
 * only check. It also enforces 2FA, which surfaces here as a 404 only after the
 * backend has already refused.
 */
export default async function PlatformLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const admin = await getPlatformAdmin();
  if (!admin) notFound();

  return <PlatformShell admin={admin}>{children}</PlatformShell>;
}
