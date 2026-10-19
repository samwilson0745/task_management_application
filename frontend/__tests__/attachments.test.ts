import { describe, it, expect, vi, afterEach } from "vitest";
import { openAttachment } from "@/lib/attachments";
import type { TaskAttachment } from "@/lib/types";

const attachment: TaskAttachment = {
  id: "a1",
  task_id: "t1",
  user_id: "u1",
  filename: "report.bin",
  content_type: "application/octet-stream",
  size_bytes: 3,
  created_at: "2024-01-01T00:00:00Z",
};

function stubBrowser(res: Response) {
  const tab = { opener: {}, location: { href: "" }, close: vi.fn() };
  const fetchMock = vi.fn().mockResolvedValue(res);
  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("open", vi.fn().mockReturnValue(tab));
  URL.createObjectURL = vi.fn().mockReturnValue("blob:x");
  URL.revokeObjectURL = vi.fn();
  return { tab, fetchMock };
}

describe("openAttachment", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("sends the token in the Authorization header, not the URL", async () => {
    const { fetchMock } = stubBrowser(
      new Response("x", { status: 200, headers: { "Content-Type": "image/png" } })
    );
    await openAttachment("t1", attachment, "secret-token");

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).not.toContain("secret-token");
    expect(init.headers.Authorization).toBe("Bearer secret-token");
  });

  it("previews safe types in the opened tab", async () => {
    const { tab } = stubBrowser(
      new Response("x", { status: 200, headers: { "Content-Type": "image/png" } })
    );
    await openAttachment("t1", attachment, "tok");
    expect(tab.location.href).toBe("blob:x");
    expect(tab.close).not.toHaveBeenCalled();
  });

  it("downloads (and never previews) other types", async () => {
    const { tab } = stubBrowser(
      new Response("<html>", { status: 200, headers: { "Content-Type": "application/octet-stream" } })
    );
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
    await openAttachment("t1", attachment, "tok");
    expect(tab.location.href).toBe("");
    expect(tab.close).toHaveBeenCalled();
    expect(click).toHaveBeenCalled();
    click.mockRestore();
  });

  it("closes the tab and throws on a failed request", async () => {
    const { tab } = stubBrowser(
      new Response(JSON.stringify({ error: "attachment not found" }), { status: 404 })
    );
    await expect(openAttachment("t1", attachment, "tok")).rejects.toThrow("attachment not found");
    expect(tab.close).toHaveBeenCalled();
  });
});
