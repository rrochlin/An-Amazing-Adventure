/**
 * api.service.ts
 * Base Axios wrappers. All authenticated calls attach the Cognito access token.
 * Token refresh is handled by Cognito SDK — no custom interceptor needed.
 */
import axios, { type AxiosRequestConfig, type AxiosResponse } from "axios";
import { getAuthHeader, refreshSession, ClearUserAuth } from "./auth.service";

export const APP_URI = import.meta.env.VITE_APP_URI || "/api";

// Single response interceptor: on 401, attempt token refresh once then give up.
axios.interceptors.response.use(
  (r) => r,
  async (error) => {
    const original = error.config as AxiosRequestConfig & { _retry?: boolean };
    if (error.response?.status === 401 && !original._retry) {
      original._retry = true;
      const refreshed = await refreshSession();
      if (refreshed) {
        original.headers = {
          ...original.headers,
          Authorization: getAuthHeader(),
        };
        return axios(original);
      }
      ClearUserAuth();
      window.location.href = "/login";
    }
    return Promise.reject(error);
  }
);

function authConfig(): AxiosRequestConfig {
  return { headers: { Authorization: getAuthHeader() } };
}

export async function GET<T>(uri: string): Promise<AxiosResponse<T>> {
  return axios.get<T>(`${APP_URI}/${uri}`, authConfig());
}

export async function POST<T>(uri: string, body?: unknown): Promise<AxiosResponse<T>> {
  return axios.post<T>(`${APP_URI}/${uri}`, body, authConfig());
}

export async function PUT<T>(uri: string, body: unknown): Promise<AxiosResponse<T>> {
  return axios.put<T>(`${APP_URI}/${uri}`, body, authConfig());
}

export async function DELETE<T>(uri: string): Promise<AxiosResponse<T>> {
  return axios.delete<T>(`${APP_URI}/${uri}`, authConfig());
}
