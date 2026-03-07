// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/services/api.service", () => ({
  GET: vi.fn(),
  POST: vi.fn(),
  PUT: vi.fn(),
  DELETE: vi.fn(),
}));

vi.mock("@/services/auth.service", () => ({
  getAuthHeader: () => "Bearer test-token",
  refreshSession: vi.fn(),
  ClearUserAuth: vi.fn(),
}));

import * as apiService from "@/services/api.service";
import { UpdateUser } from "@/services/api.users";

const mockPut = apiService.PUT as ReturnType<typeof vi.fn>;

beforeEach(() => vi.clearAllMocks());

describe("UpdateUser", () => {
  it("calls PUT api/users with body and returns success", async () => {
    mockPut.mockResolvedValueOnce({ data: { status: "ok" }, status: 200 });
    const result = await UpdateUser({ email: "new@example.com" });
    expect(mockPut).toHaveBeenCalledWith("api/users", { email: "new@example.com" });
    expect(result.success).toBe(true);
  });

  it("returns success:false on error", async () => {
    mockPut.mockRejectedValueOnce(new Error("Unauthorized"));
    const result = await UpdateUser({ email: "x@y.com" });
    expect(result.success).toBe(false);
  });

  it("can call UpdateUser with empty body", async () => {
    mockPut.mockResolvedValueOnce({ data: {}, status: 200 });
    const result = await UpdateUser({});
    expect(result.success).toBe(true);
  });
});
