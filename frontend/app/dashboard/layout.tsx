import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { getServerUser, getServerWorkspaces } from "@/lib/server-auth";
import { AppShell } from "@/components/app/app-shell";

export const metadata: Metadata = { title: "Dashboard" };

/**
 * Server-side guard for the whole authenticated app. Runs before any UI renders:
 * not signed in → /login; signed in but no workspace yet → /onboarding.
 */
export default async function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const user = await getServerUser();
  if (!user) redirect("/login");

  const workspaces = await getServerWorkspaces();
  if (workspaces.length === 0) redirect("/onboarding");

  return (
    <AppShell user={user} workspaces={workspaces}>
      {children}
    </AppShell>
  );
}
