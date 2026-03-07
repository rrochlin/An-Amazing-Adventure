// @vitest-environment jsdom
import { describe, it, expect, beforeEach, vi } from "vitest";
import {
  getStoredTokens,
  isAuthenticated,
  ClearUserAuth,
  getUserSub,
} from "@/services/auth.service";

vi.mock("amazon-cognito-identity-js", () => ({
  CognitoUserPool: vi.fn(),
  CognitoUser: vi.fn(() => ({
    authenticateUser: vi.fn(),
    forgotPassword: vi.fn(),
    confirmPassword: vi.fn(),
    confirmRegistration: vi.fn(),
    resendConfirmationCode: vi.fn(),
    signOut: vi.fn(),
    getSession: vi.fn(),
  })),
  AuthenticationDetails: vi.fn(),
  CognitoUserAttribute: vi.fn(),
}));

const STORAGE_KEY = "AAA_TOKENS";

const validTokens = {
  accessToken: "access.token.here",
  idToken: "id.token.here",
  refreshToken: "refresh.token.here",
  expiresAt: Date.now() + 60_000, // 1 min in the future
  userSub: "user-uuid-123",
};

const expiredTokens = {
  ...validTokens,
  expiresAt: Date.now() - 1000, // expired
};

beforeEach(() => {
  localStorage.clear();
});

describe("getStoredTokens", () => {
  it("returns null when no tokens stored", () => {
    expect(getStoredTokens()).toBeNull();
  });

  it("returns parsed tokens when stored", () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(validTokens));
    const tokens = getStoredTokens();
    expect(tokens?.userSub).toBe("user-uuid-123");
    expect(tokens?.accessToken).toBe("access.token.here");
  });

  it("returns null for corrupt JSON", () => {
    localStorage.setItem(STORAGE_KEY, "not-json{");
    expect(getStoredTokens()).toBeNull();
  });
});

describe("isAuthenticated", () => {
  it("returns false when no tokens", () => {
    expect(isAuthenticated()).toBe(false);
  });

  it("returns true for valid non-expired tokens", () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(validTokens));
    expect(isAuthenticated()).toBe(true);
  });

  it("returns false for expired tokens", () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(expiredTokens));
    expect(isAuthenticated()).toBe(false);
  });
});

describe("ClearUserAuth", () => {
  it("removes tokens from localStorage", () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(validTokens));
    ClearUserAuth();
    expect(localStorage.getItem(STORAGE_KEY)).toBeNull();
  });

  it("does not throw when nothing is stored", () => {
    expect(() => ClearUserAuth()).not.toThrow();
  });
});

describe("getUserSub", () => {
  it("returns empty string when not authenticated", () => {
    expect(getUserSub()).toBe("");
  });

  it("returns the sub from stored tokens", () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(validTokens));
    expect(getUserSub()).toBe("user-uuid-123");
  });
});
