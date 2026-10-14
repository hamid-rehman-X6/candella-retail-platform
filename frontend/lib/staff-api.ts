import { apiFetch, apiGet, apiPost } from "@/lib/api";

export type StaffMember = {
  userId: string;
  email: string;
  fullName: string;
  role: "owner" | "admin" | "member";
  status: "active" | "suspended";
};

export type StaffInvitation = {
  id: string;
  email: string;
  role: "owner" | "admin" | "member";
  status: "pending";
  expiresAt: string;
};

export const listStaff = (tenantId: string) =>
  apiGet<{ members: StaffMember[]; invitations: StaffInvitation[] }>(
    "/staff",
    tenantId,
  );

export const inviteStaff = (
  tenantId: string,
  email: string,
  role: StaffInvitation["role"],
) =>
  apiPost<{ invitation: StaffInvitation }>(
    "/staff/invitations",
    { email, role },
    tenantId,
  );

export const resendStaffInvite = (tenantId: string, id: string) =>
  apiPost<{ invitation: StaffInvitation }>(
    `/staff/invitations/${encodeURIComponent(id)}/resend`,
    {},
    tenantId,
  );

export const revokeStaffInvite = (tenantId: string, id: string) =>
  apiFetch<{ revoked: boolean }>(
    `/staff/invitations/${encodeURIComponent(id)}`,
    { method: "DELETE", tenantId },
  );

export const changeStaffRole = (
  tenantId: string,
  userId: string,
  role: StaffMember["role"],
) =>
  apiFetch<{ updated: boolean }>(`/staff/${encodeURIComponent(userId)}`, {
    method: "PATCH",
    body: JSON.stringify({ role }),
    tenantId,
  });

export const removeStaff = (tenantId: string, userId: string) =>
  apiFetch<{ removed: boolean }>(`/staff/${encodeURIComponent(userId)}`, {
    method: "DELETE",
    tenantId,
  });

export const acceptStaffInvitation = (token: string) =>
  apiPost<{ tenantId: string }>("/invitations/accept", { token });
