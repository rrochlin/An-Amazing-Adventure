/*
	PUT /api/users
	POST /api/users
 */

import type { AxiosRequestConfig } from "axios";
import type {
	ApiCreateUserRequest,
	ApiCreateUserResponse,
	ApiLoginRequest,
	ApiLoginResponse,
	ApiUpdateUserRequest,
	ApiUpdateUserResponse,
	CreateUserResponse,
	LoginResponse,
	UpdateUserResponse,
} from "../types/api.types";
import { getAuthHeaders } from "./auth.service";
import { POST } from "./api.service";
import type { stored_tokens } from "../types/types";

export async function Login(data: ApiLoginRequest): Promise<LoginResponse> {
	// Login will handle saving tokens to local storage
	console.log("trying to log in with credentials")
	const config: AxiosRequestConfig = {
		headers: await getAuthHeaders(),
		data: data,
	};
	const response = await POST<ApiLoginResponse>("login", config);
	console.log("received response", response)
	if (response.status > 299) {
		console.error(`login failed ${response.data}`);
		return { success: false };
	}
	const localCreds: stored_tokens = {
		jwt: response.data.token!,
		rtoken: response.data.refresh_token!,
		expiresAt: new Date(response.data.updated_at.valueOf() + 60 * 1000),
	};
	localStorage.setItem("AAA_JWT", JSON.stringify(localCreds));
	return { success: true };
}

export async function CreateNewUser(
	data: ApiCreateUserRequest,
): Promise<CreateUserResponse> {
	const config: AxiosRequestConfig = {
		headers: await getAuthHeaders(),
		data: data,
	};
	const response = await POST<ApiCreateUserResponse>("users", config);
	if (response.status != 201) {
		console.error(`user creation failed ${response.data}`);
		return { success: false };
	}
	const localCreds: stored_tokens = {
		jwt: response.data.token!,
		rtoken: response.data.refresh_token!,
		expiresAt: new Date(response.data.updated_at.valueOf() + 60 * 1000),
	};
	localStorage.setItem("AAA_JWT", JSON.stringify(localCreds));
	return { success: true };
}

export async function UpdateUser(
	data: ApiUpdateUserRequest,
): Promise<UpdateUserResponse> {
	const config: AxiosRequestConfig = {
		headers: await getAuthHeaders(),
		data: data,
	};
	const response = await POST<ApiUpdateUserResponse>("users", config);
	if (response.status != 201) {
		console.error(`user creation failed ${response.data}`);
		return { success: false };
	}
	const localCreds: stored_tokens = {
		jwt: response.data.token!,
		rtoken: response.data.refresh_token!,
		expiresAt: new Date(response.data.updated_at.valueOf() + 60 * 1000),
	};
	localStorage.setItem("AAA_JWT", JSON.stringify(localCreds));
	return { success: true };
}
