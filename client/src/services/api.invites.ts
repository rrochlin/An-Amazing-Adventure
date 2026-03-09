import { GET, POST } from "./api.service";
import type { InviteInfo, CreateInviteResponse } from "../types/types";

export interface CreateInviteParams {
  session_id: string;
  max_uses?: number; // default 10
  ttl_days?: number; // default 7
}

export interface JoinInviteResponse {
  session_id: string;
}

export async function CreateInvite(
  params: CreateInviteParams,
): Promise<CreateInviteResponse> {
  const res = await POST<CreateInviteResponse>("api/invites", params);
  return res.data;
}

export async function GetInvite(code: string): Promise<InviteInfo> {
  const res = await GET<InviteInfo>(`api/invites/${code}`);
  return res.data;
}

export async function JoinInvite(code: string): Promise<JoinInviteResponse> {
  const res = await POST<JoinInviteResponse>(`api/invites/${code}/join`, {});
  return res.data;
}
