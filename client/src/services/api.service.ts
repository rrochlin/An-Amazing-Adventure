import axios, { type AxiosRequestConfig, type AxiosResponse } from "axios";
import { getAuthHeaders } from "./auth.service";

const APP_URI = import.meta.env.VITE_APP_URI || "http://localhost:8080";

export async function GET<T>(uri: string): Promise<AxiosResponse<T>> {
	const config: AxiosRequestConfig = {
		headers: await getAuthHeaders(),
	};
	const response: AxiosResponse<T> = await axios.get(
		`${APP_URI}/${uri}`,
		config,
	);
	if (response.status > 299) {
		console.error("server returned error response", response);
	}
	return response;
}

export async function POST<T>(
	uri: string,
	body?: any,
): Promise<AxiosResponse<T>> {
	const config: AxiosRequestConfig = {
		headers: await getAuthHeaders(),
	};
	const response: AxiosResponse<T> = await axios.post(
		`${APP_URI}/${uri}`,
		body,
		config,
	);
	if (response.status > 299) {
		console.error("server returned error response", response);
	}
	return response;
}

export async function PUT<T>(
	uri: string,
	body: any,
): Promise<AxiosResponse<T>> {
	const config: AxiosRequestConfig = {
		headers: await getAuthHeaders(),
	};
	const response: AxiosResponse<T> = await axios.put(
		`${APP_URI}/${uri}`,
		body,
		config,
	);
	if (response.status > 299) {
		console.error("server returned error response", response);
	}
	return response;
}
