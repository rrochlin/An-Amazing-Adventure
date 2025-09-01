import {
  AxiosHeaders,
  AxiosRequestConfig,
  AxiosRequestHeaders,
  AxiosResponse,
} from "axios";
import {
  ApiRefreshResponse,
  ApiRevokeResponse,
  RefreshResponse,
  RevokeResponse,
} from "./api.types";
import { POST } from "./api.service";
import { stored_tokens } from "../models";

export async function getAuthHeaders(
  rtoken?: boolean,
): Promise<AxiosRequestHeaders> {
  let raw_localJWT = localStorage.getItem("AAA_JWT");
  const headers = new AxiosHeaders();
  if (!raw_localJWT) {
    console.error("unable to retrieve JWT, please sign in");
    return headers;
  }
  const localJWT: stored_tokens = JSON.parse(raw_localJWT);
  if (localJWT.expiresAt > new Date()) {
    console.log("refresh token has expired user will need to reauth");
    const refreshRes = await refreshToken();
    if (refreshRes.status != 200)
      throw Error("user must re-authenticate fully");
    localStorage.setItem("AAA_JWT", JSON.stringify(localJWT));
  }
  if (rtoken) {
    headers.setAuthorization(`Bearer ${localJWT.rtoken}`);
  } else {
    headers.setAuthorization(`Bearer ${localJWT.jwt}`);
  }

  return headers;
}

export async function refreshToken(): Promise<AxiosResponse<RefreshResponse>> {
  // just for this
  const config: AxiosRequestConfig = { headers: await getAuthHeaders(true) };
  const response = await POST<ApiRefreshResponse>("refresh", config);
  return response;
}

export async function revokeToken(): Promise<RevokeResponse> {
  const config: AxiosRequestConfig = { headers: await getAuthHeaders(true) };
  const response = await POST<ApiRevokeResponse>("revoke", config);
  return { success: response.status == 204 };
}
