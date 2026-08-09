import { serverFetch } from "@/lib/server-auth";

/**
 * Server-only client for the platform console.
 *
 * Every read is done from a Server Component: the console renders across all
 * tenants, so keeping the data on the server means cross-tenant records never
 * pass through a browser bundle or a client-side cache.
 */

export type PlatformAdmin = {
  id: string;
  email: string;
  fullName: string;
  role: "owner" | "support" | "billing" | "engineering";
};

export type StatusCounts = {
  total: number;
  active: number;
  trialing: number;
  suspended: number;
  cancelled: number;
};

export type ActivityEntry = {
  id: string;
  tenantId: string | null;
  tenantName: string | null;
  actorType: "user" | "platform_admin" | "system";
  actorEmail: string | null;
  action: string;
  entityType: string;
  entityId: string;
  ipAddress: string | null;
  createdAt: string;
};

export type PlatformOverview = {
  tenants: StatusCounts;
  totalUsers: number;
  newUsers7d: number;
  newTenants7d: number;
  activeSessions: number;
  recentActivity: ActivityEntry[];
};

export type PlatformTenant = {
  id: string;
  name: string;
  slug: string;
  industryType: string;
  status: string;
  currencyCode: string;
  memberCount: number;
  ownerEmail: string;
  createdAt: string;
};

export type TenantMember = {
  userId: string;
  email: string;
  fullName: string;
  role: string;
  status: string;
  mfaEnabled: boolean;
  lastLoginAt: string | null;
  joinedAt: string;
};

export type PlatformTenantDetail = PlatformTenant & {
  timezone: string;
  members: TenantMember[];
  recentActivity: ActivityEntry[];
};

/** The signed-in admin, or null when the user isn't staff (backend answers 404). */
export async function getPlatformAdmin(): Promise<PlatformAdmin | null> {
  const data = await serverFetch<{ admin: PlatformAdmin }>("/platform/me");
  return data?.admin ?? null;
}

export async function getPlatformOverview(): Promise<PlatformOverview | null> {
  const data = await serverFetch<{ overview: PlatformOverview }>(
    "/platform/overview",
  );
  return data?.overview ?? null;
}

export async function getPlatformTenants(params: {
  search?: string;
  status?: string;
  industry?: string;
  limit?: number;
  offset?: number;
}): Promise<{ tenants: PlatformTenant[]; total: number }> {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== "") query.set(key, String(value));
  }
  const data = await serverFetch<{ tenants: PlatformTenant[]; total: number }>(
    `/platform/tenants?${query.toString()}`,
  );
  return data ?? { tenants: [], total: 0 };
}

export async function getPlatformTenant(
  tenantId: string,
): Promise<PlatformTenantDetail | null> {
  const data = await serverFetch<{ tenant: PlatformTenantDetail }>(
    `/platform/tenants/${tenantId}`,
  );
  return data?.tenant ?? null;
}
