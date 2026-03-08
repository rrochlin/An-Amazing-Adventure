import { GET, PUT } from "./api.service";

export interface AdminUserView {
  user_id: string;
  email: string;
  role: "admin" | "user" | "restricted";
  ai_enabled: boolean;
  token_limit: number; // 0 = unlimited
  tokens_used: number;
  games_limit: number; // 0 = unlimited
  billing_mode: string;
  notes?: string;
  created_at: number;
}

export interface AdminStats {
  total_users: number;
  admin_users: number;
  approved_users: number;
  restricted_users: number;
  total_tokens_used: number;
}

export interface UpdateUserRequest {
  role: "admin" | "user" | "restricted";
  ai_enabled: boolean;
  token_limit: number;
  games_limit: number;
  notes: string;
}

export async function listAdminUsers(): Promise<AdminUserView[]> {
  const res = await GET<AdminUserView[]>("api/admin/users");
  return res.data;
}

export async function updateAdminUser(
  userId: string,
  data: UpdateUserRequest,
): Promise<void> {
  await PUT(`api/admin/users/${userId}`, data);
}

export async function getAdminStats(): Promise<AdminStats> {
  const res = await GET<AdminStats>("api/admin/stats");
  return res.data;
}
