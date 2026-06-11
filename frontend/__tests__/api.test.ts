import { describe, it, expect, vi, afterEach } from "vitest";
import { apiRequest, ApiError } from "@/lib/api";

describe("apiRequest", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("returns parsed JSON on success", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: "1", title: "Test task" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    const result = await apiRequest<{ id: string; title: string }>("/tasks/1", { token: "abc" });

    expect(result).toEqual({ id: "1", title: "Test task" });
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/tasks/1"),
      expect.objectContaining({
        headers: expect.objectContaining({ Authorization: "Bearer abc" }),
      })
    );
  });

  it("throws ApiError with message and details on failure", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({ error: "validation failed", details: { title: "is required" } }),
        { status: 422, headers: { "Content-Type": "application/json" } }
      )
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(apiRequest("/tasks/", { method: "POST", body: {} })).rejects.toMatchObject({
      message: "validation failed",
      status: 422,
      details: { title: "is required" },
    });
  });

  it("returns undefined for 204 No Content responses", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await apiRequest<void>("/tasks/1", { method: "DELETE" });
    expect(result).toBeUndefined();
  });

  it("ApiError is an instance of Error", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ error: "not found" }), { status: 404 })
    );
    vi.stubGlobal("fetch", fetchMock);

    try {
      await apiRequest("/tasks/missing");
      expect.fail("expected apiRequest to throw");
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      expect(err).toBeInstanceOf(Error);
    }
  });
});
