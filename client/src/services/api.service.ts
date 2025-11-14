import axios, { type AxiosRequestConfig, type AxiosResponse, type AxiosError, type InternalAxiosRequestConfig } from "axios";
import { getAuthHeaders, refreshToken, ClearUserAuth, getJWT } from "./auth.service";

const APP_URI = import.meta.env.VITE_APP_URI || "http://localhost:8080";

// Track if we're currently refreshing to avoid multiple refresh calls
let isRefreshing = false;
let failedQueue: Array<{
  resolve: (value?: any) => void;
  reject: (reason?: any) => void;
}> = [];

const processQueue = (error: Error | null, token: string | null = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve(token);
    }
  });
  failedQueue = [];
};

// Axios response interceptor for handling 401 errors
axios.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean };

    // If error is 401 and we haven't retried yet
    if (error.response?.status === 401 && !originalRequest._retry) {
      if (isRefreshing) {
        // If already refreshing, queue this request
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject });
        })
          .then(() => {
            // Retry with new token
            return axios(originalRequest);
          })
          .catch((err) => {
            return Promise.reject(err);
          });
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        const tokens = getJWT();
        if (!tokens) {
          throw new Error("No tokens available");
        }

        // Attempt to refresh the token
        const refreshHeaders = new (await import("axios")).AxiosHeaders();
        refreshHeaders.setAuthorization(`Bearer ${tokens.rtoken}`);

        await refreshToken(refreshHeaders);

        // Token refreshed successfully
        processQueue(null, tokens.jwt);
        isRefreshing = false;

        // Retry the original request with new token
        originalRequest.headers = await getAuthHeaders();
        return axios(originalRequest);
      } catch (refreshError) {
        // Refresh failed - clear auth and redirect to login
        processQueue(refreshError as Error, null);
        isRefreshing = false;
        ClearUserAuth();

        // Redirect to login
        window.location.href = `/login?redirect=${encodeURIComponent(window.location.pathname)}`;
        return Promise.reject(refreshError);
      }
    }

    return Promise.reject(error);
  }
);

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

export async function DELETE<T>(uri: string): Promise<AxiosResponse<T>> {
  const config: AxiosRequestConfig = {
    headers: await getAuthHeaders(),
  };
  const response: AxiosResponse<T> = await axios.delete(
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
