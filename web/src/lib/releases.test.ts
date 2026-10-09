import { afterEach, describe, expect, it, vi } from "vite-plus/test";
import {
  compareSemver,
  fetchLatestRelease,
  isNewerVersion,
  normalizeVersion,
  parseReleaseNotes,
  releaseExcerpt,
} from "./releases";

afterEach(() => {
  vi.restoreAllMocks();
});

describe("compareSemver", () => {
  it("orders dotted versions", () => {
    expect(compareSemver("0.1.0", "0.1.0")).toBe(0);
    expect(compareSemver("0.1.0", "0.2.0")).toBeLessThan(0);
    expect(compareSemver("v0.2.0", "0.1.9")).toBeGreaterThan(0);
    expect(isNewerVersion("0.2.0", "0.1.0")).toBe(true);
    expect(isNewerVersion("0.1.0", "0.1.0")).toBe(false);
  });

  it("orders prereleases and ignores metadata and invalid versions", () => {
    const ordered = [
      "1.0.0-alpha",
      "1.0.0-alpha.1",
      "1.0.0-alpha.beta",
      "1.0.0-beta",
      "1.0.0-beta.2",
      "1.0.0-beta.11",
      "1.0.0-rc.1",
      "1.0.0",
    ];
    for (let i = 1; i < ordered.length; i++)
      expect(isNewerVersion(ordered[i], ordered[i - 1])).toBe(true);
    expect(compareSemver("v0.3.0+build.1", "0.3.0+build.2")).toBe(0);
    for (const version of ["dev", "", "0.3", "01.3.0", "1.0.0-01"]) {
      expect(isNewerVersion("0.15.0", version)).toBe(false);
      expect(isNewerVersion(version, "0.3.0")).toBe(false);
    }
  });
  it("normalizes leading v", () => {
    expect(normalizeVersion("v1.2.3")).toBe("1.2.3");
  });
});

describe("releaseExcerpt", () => {
  it("strips markdown and truncates", () => {
    const body = "## Highlights\n\n- **Fix** something\n\n```go\nfmt.Println()\n```";
    const excerpt = releaseExcerpt(body, 40);
    expect(excerpt).not.toContain("```");
    expect(excerpt.length).toBeLessThanOrEqual(41);
  });
});

describe("parseReleaseNotes", () => {
  const body = [
    "## [0.7.0](https://github.com/x/y/compare/v0.6.1...v0.7.0) (2026-08-09)",
    "",
    "### Features",
    "",
    "* **mobile:** option call/put chips ([#169](https://github.com/x/y/issues/169)) ([ba6e999](https://github.com/x/y/commit/ba6e999a80be1e7e0339071ab948a07bb5900a26))",
    "",
    "### Bug Fixes",
    "",
    "* **docker:** build the web bundle natively ([#171](https://github.com/x/y/issues/171)) ([b3fe070](https://github.com/x/y/commit/b3fe0706))",
    "* plain fix without scope or refs",
  ].join("\n");

  it("parses release-please sections, scopes, and PR refs", () => {
    const sections = parseReleaseNotes(body);
    expect(sections.map((s) => s.title)).toEqual(["Features", "Bug Fixes"]);
    expect(sections[0].items).toEqual([
      {
        scope: "mobile",
        text: "option call/put chips",
        prLabel: "#169",
        prUrl: "https://github.com/x/y/issues/169",
      },
    ]);
    expect(sections[1].items[1]).toEqual({
      scope: null,
      text: "plain fix without scope or refs",
      prLabel: null,
      prUrl: null,
    });
  });

  it("drops commit-sha refs but keeps the PR link", () => {
    const [fixes] = parseReleaseNotes(body).slice(1);
    expect(fixes.items[0]).toEqual({
      scope: "docker",
      text: "build the web bundle natively",
      prLabel: "#171",
      prUrl: "https://github.com/x/y/issues/171",
    });
  });

  it("returns no sections for freeform bodies", () => {
    expect(parseReleaseNotes("Bug fixes and polish.")).toEqual([]);
    expect(parseReleaseNotes("")).toEqual([]);
  });

  it("flattens inline links left in item text", () => {
    const sections = parseReleaseNotes(
      "### Features\n* see [the docs](https://example.com) for details",
    );
    expect(sections[0].items[0].text).toBe("see the docs for details");
  });
});

describe("fetchLatestRelease", () => {
  it("parses GitHub latest release payload", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(
        JSON.stringify({
          tag_name: "v0.2.0",
          name: "TraderMemos 0.2.0",
          body: "Bug fixes and polish.",
          html_url: "https://github.com/HoesenBruce/TradeLens/releases/tag/v0.2.0",
          published_at: "2026-07-01T12:00:00Z",
          prerelease: false,
        }),
        { status: 200 },
      ),
    );
    await expect(fetchLatestRelease()).resolves.toEqual({
      version: "0.2.0",
      tag: "v0.2.0",
      name: "TraderMemos 0.2.0",
      body: "Bug fixes and polish.",
      excerpt: "Bug fixes and polish.",
      publishedAt: "2026-07-01T12:00:00Z",
      url: "https://github.com/HoesenBruce/TradeLens/releases/tag/v0.2.0",
      prerelease: false,
    });
  });

  it("uses TradeLens even when upstream has v0.15.0", async () => {
    const request = vi.spyOn(globalThis, "fetch").mockImplementation(
      async (url) =>
        new Response(
          JSON.stringify({
            tag_name: String(url).includes("HoesenBruce/TradeLens") ? "v0.3.0" : "v0.15.0",
          }),
        ),
    );
    const latest = await fetchLatestRelease();
    expect(latest?.version).toBe("0.3.0");
    expect(isNewerVersion(latest!.version, "0.3.0")).toBe(false);
    expect(request).toHaveBeenCalledWith(
      "https://api.github.com/repos/HoesenBruce/TradeLens/releases/latest",
      expect.objectContaining({ cache: "no-store" }),
    );
  });
  it.each([
    { tag_name: "v0.4.0-rc.1" },
    { tag_name: "v0.4.0", prerelease: true },
    { tag_name: "v0.4.0", draft: true },
    { tag_name: "garbage" },
  ])("ignores non-stable payload %j", async (payload) => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify(payload)));
    await expect(fetchLatestRelease()).resolves.toBeNull();
  });
  it("reports offline failure", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValue(new TypeError("Failed to fetch"));
    await expect(fetchLatestRelease()).rejects.toThrow("Failed to fetch");
  });

  it("reports API failure", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response("", { status: 503 }));
    await expect(fetchLatestRelease()).rejects.toThrow("Release check failed (503)");
  });
  it("returns null when no releases exist", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response("", { status: 404 }));
    await expect(fetchLatestRelease()).resolves.toBeNull();
  });
});
