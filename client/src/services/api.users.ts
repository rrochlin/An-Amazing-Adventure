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
import { PUT } from "./api.service";
import type { stored_tokens } from "../types/types";
import axios from "axios";


const BackendURI = import.meta.env.VITE_APP_URI;

export async function Login(body: ApiLoginRequest): Promise<LoginResponse> {
	// Login will handle saving tokens to local storage
	console.log("trying to log in with credentials")
	const response = await axios.post<ApiLoginResponse>(`${BackendURI}/login`, body);
	console.log("received response", response)
	if (response.status > 299) {
		console.error(`login failed ${response.data}`);
		return { success: false };
	}
	const localCreds: stored_tokens = {
		jwt: response.data.token!,
		rtoken: response.data.refresh_token!,
		expiresAt: Date.now() + 60 * 100_000,
	};
	localStorage.setItem("AAA_JWT", JSON.stringify(localCreds));
	return { success: true };
}

export async function CreateNewUser(
	body: ApiCreateUserRequest,
): Promise<CreateUserResponse> {
	const response = await axios.post<ApiCreateUserResponse>(`${BackendURI}/users`, body);
	if (response.status != 201) {
		console.error(`user creation failed ${response.data}`);
		return { success: false };
	}
	const localCreds: stored_tokens = {
		jwt: response.data.token!,
		rtoken: response.data.refresh_token!,
		expiresAt: Date.now() + 60 * 100_000,
	};
	localStorage.setItem("AAA_JWT", JSON.stringify(localCreds));
	return { success: true };
}

export async function UpdateUser(
	body: ApiUpdateUserRequest,
): Promise<UpdateUserResponse> {
	const response = await PUT<ApiUpdateUserResponse>("users", body);
	if (response.status != 201) {
		console.error(`user creation failed ${response.data}`);
		return { success: false };
	}
	const localCreds: stored_tokens = {
		jwt: response.data.token!,
		rtoken: response.data.refresh_token!,
		expiresAt: Date.now() + 60 * 100_000,
	};
	localStorage.setItem("AAA_JWT", JSON.stringify(localCreds));
	return { success: true };
}
