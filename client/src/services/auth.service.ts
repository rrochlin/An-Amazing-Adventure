import type {
	AxiosRequestConfig,
	AxiosRequestHeaders,
	AxiosResponse,
} from "axios";
import { AxiosHeaders } from "axios";
import type {
	ApiRefreshResponse,
	ApiRevokeResponse,
	RefreshResponse,
	RevokeResponse,
} from "../types/api.types";
import { POST } from "./api.service";
import type { stored_tokens } from "../types/types";
import { redirect } from "@tanstack/react-router";

export async function getAuthHeaders(
	rtoken?: boolean,
): Promise<AxiosRequestHeaders> {
	const headers = new AxiosHeaders();
	const localJWT: stored_tokens = getJWT()
	if (localJWT.expiresAt < Date.now()) {
		console.log("refresh token has expired user will need to reauth");
		throw redirect({ to: "/login", search: { redirect: location.href } })
	}
	if (rtoken) {
		headers.setAuthorization(`Bearer ${localJWT.rtoken}`);
	} else {
		headers.setAuthorization(`Bearer ${localJWT.jwt}`);
	}

	return headers;
}

function getJWT(): stored_tokens {
	let raw_localJWT = localStorage.getItem("AAA_JWT");

	if (!raw_localJWT) {
		console.error("unable to retrieve JWT, please sign in");
	}
	const localJWT: stored_tokens = JSON.parse(raw_localJWT ?? "");
	return localJWT
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

export function isAuthenticated(): boolean {
	const tokens = getJWT()
	if (!tokens) return false
	return tokens.expiresAt > Date.now()
}
