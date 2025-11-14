import type {
  AxiosRequestConfig,
  AxiosRequestHeaders,
  AxiosResponse,
} from "axios";
import axios, { AxiosHeaders } from "axios";
import type {
  ApiRefreshResponse,
  ApiRevokeResponse,
  RefreshResponse,
  RevokeResponse,
} from "../types/api.types";
import type { stored_tokens } from "../types/types";
import { redirect } from "@tanstack/react-router";

const APP_URI = import.meta.env.VITE_APP_URI || "http://localhost:8080";

export async function getAuthHeaders(
  rtoken?: boolean,
): Promise<AxiosRequestHeaders> {
  const headers = new AxiosHeaders();
  const localJWT = getJWT();

  if (!localJWT) {
    console.error("No JWT found, redirecting to login");
    throw redirect({ to: "/login", search: { redirect: location.href } });
  }

  // Check if refresh token itself has expired
  if (localJWT.expiresAt < Date.now()) {
    console.log("Refresh token has expired, user will need to reauth");
    ClearUserAuth();
    throw redirect({ to: "/login", search: { redirect: location.href } });
  }

  // Set the appropriate token in headers
  if (rtoken) {
    headers.setAuthorization(`Bearer ${localJWT.rtoken}`);
  } else {
    headers.setAuthorization(`Bearer ${localJWT.jwt}`);
  }

  return headers;
}

export function getJWT(): stored_tokens | undefined {
  let raw_localJWT = localStorage.getItem("AAA_JWT");

  if (!raw_localJWT) {
    console.error("unable to retrieve JWT, please sign in");
    return undefined;
  }
  const localJWT: stored_tokens = JSON.parse(raw_localJWT ?? "");
  return localJWT;
}

export async function refreshToken(
  headers: AxiosRequestHeaders,
): Promise<AxiosResponse<RefreshResponse>> {
  // Use axios directly to avoid circular dependency and interceptor triggering
  const config: AxiosRequestConfig = { headers: headers };
  const response = await axios.post<ApiRefreshResponse>(
    `${APP_URI}/api/refresh`,
    {},
    config
  );

  let raw_localJWT = localStorage.getItem("AAA_JWT");
  const localJWT: stored_tokens = JSON.parse(raw_localJWT ?? "");
  localJWT.expiresAt = Date.now();
  localJWT.rtoken = response.data.token;
  localStorage.setItem("AAA_JWT", JSON.stringify(localJWT));
  return response;
}

export async function revokeToken(): Promise<RevokeResponse> {
  const config: AxiosRequestConfig = { headers: await getAuthHeaders(true) };
  const response = await axios.post<ApiRevokeResponse>(
    `${APP_URI}/api/revoke`,
    {},
    config
  );
  return { success: response.status == 204 };
}

export function isAuthenticated(): boolean {
  const tokens = getJWT();
  if (!tokens) return false;
  return tokens.expiresAt > Date.now();
}

export function ClearUserAuth() {
  localStorage.removeItem("AAA_JWT");
}
