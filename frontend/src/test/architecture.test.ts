import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative, resolve } from "node:path";
import { describe, expect, it } from "vitest";

const srcRoot = resolve(import.meta.dirname, "..");

/** The only module allowed to reach the generated Wails bindings. */
const bridgeAdapter = "infrastructure/bridge/applicationInfoBridge.ts";

/** The only directory allowed to hold raw colour values. */
const tokenDirectory = "ui/tokens/";

function sourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) {
      return sourceFiles(full);
    }
    return /\.(ts|tsx)$/.test(entry) ? [full] : [];
  });
}

const files = sourceFiles(srcRoot)
  .map((file) => ({
    path: relative(srcRoot, file).split("\\").join("/"),
    text: readFileSync(file, "utf8"),
  }))
  // The check inspects shipped application code only. Test helpers and the
  // boundary test itself legitimately mention the generated bindings path.
  .filter((file) => !file.path.startsWith("test/") && !/\.test\.tsx?$/.test(file.path));

/**
 * A single source of truth for the "no HTTP transport" rule. It is a regression
 * check over the known forbidden transport paths, not a proof that no network
 * client whatsoever can exist: only the listed constructs are detected. `fetch`
 * is matched with its call parenthesis, optionally spaced, so that `refetch(`
 * and `prefetchQuery(` stay allowed while `fetch (` does not.
 */
const forbiddenTransport = [
  /\bfetch\s*\(/,
  /\bXMLHttpRequest\b/,
  /\baxios\b/,
  /\bWebSocket\b/,
  /\bEventSource\b/,
  /\bnavigator\s*\.\s*sendBeacon\b/,
  /["']node:https?["']/,
  /\bhttps?\s*\.\s*(request|get)\s*\(/,
  /graphql/i,
];

function usesHttpTransport(source: string): boolean {
  return forbiddenTransport.some((pattern) => pattern.test(source));
}

describe("architecture boundaries", () => {
  it("finds application sources to inspect", () => {
    expect(files.length).toBeGreaterThan(10);
  });

  it("lets only the infrastructure bridge adapter import the generated Wails bindings", () => {
    const offenders = files
      .filter((file) => /wailsjs/.test(file.text))
      .map((file) => file.path)
      .filter((path) => path !== bridgeAdapter);

    expect(offenders).toEqual([]);
  });

  it("keeps the generated bindings import inside the adapter", () => {
    const adapter = files.find((file) => file.path === bridgeAdapter);
    expect(adapter?.text).toContain("../../../wailsjs/go/desktop/Bridge");
  });

  it("keeps feature and UI components free of generated bindings and of the adapter", () => {
    const components = files.filter(
      (file) => file.path.startsWith("features/") || file.path.startsWith("ui/"),
    );

    expect(components.length).toBeGreaterThan(5);
    expect(
      components.filter((file) => /wailsjs|infrastructure\//.test(file.text)).map((f) => f.path),
    ).toEqual([]);
  });

  it("keeps the application layer independent of the infrastructure layer", () => {
    const applicationLayer = files.filter((file) => file.path.startsWith("application/"));

    expect(applicationLayer.length).toBeGreaterThan(2);
    expect(
      applicationLayer
        .filter((file) => /infrastructure\/|wailsjs/.test(file.text))
        .map((f) => f.path),
    ).toEqual([]);
  });

  it("detects every forbidden transport it is meant to reject", () => {
    const forbidden = [
      'fetch("/api")',
      'fetch ("/api")',
      "await fetch(url)",
      "new XMLHttpRequest()",
      'new WebSocket("ws://example")',
      'import axios from "axios";',
      "const query = graphql`{ me }`;",
      "useGraphQL()",
      'new EventSource("/events")',
      "navigator.sendBeacon(url, body)",
      'import http from "node:http";',
      'import https from "node:https";',
      "const req = http.request(options);",
      'http.get("http://example");',
      "https.request(options);",
      'https.get("https://example");',
    ];
    const allowed = [
      "info.refetch()",
      "const refetchOnMount = false;",
      "prefetchQuery(client)",
      "type Fetcher = () => Promise<void>;",
      "const createFetcher = () => undefined;",
      "queryClient.prefetchQuery(options)",
    ];

    expect(forbidden.filter((source) => !usesHttpTransport(source))).toEqual([]);
    expect(allowed.filter(usesHttpTransport)).toEqual([]);
  });

  it("uses no HTTP transport in application code", () => {
    const offenders = files.filter((file) => usesHttpTransport(file.text)).map((file) => file.path);

    expect(offenders).toEqual([]);
  });

  it("keeps raw colour values inside the token layer", () => {
    const offenders = files
      .filter((file) => !file.path.startsWith(tokenDirectory))
      .filter((file) => /#[0-9a-f]{3,8}\b|\brgba?\(/i.test(file.text))
      .map((file) => file.path);

    expect(offenders).toEqual([]);
  });
});

/**
 * The accepted `skipLibCheck` exception is scoped to the toolchain configuration
 * project only. This suite fails if it ever widens to application sources.
 */
const tsconfigPolicy = [
  { file: "tsconfig.json", strict: true, skipLibCheck: false },
  { file: "tsconfig.node.json", strict: true, skipLibCheck: true },
] as const;

describe("TypeScript project policy", () => {
  for (const { file, strict, skipLibCheck } of tsconfigPolicy) {
    it(`keeps strict and skipLibCheck explicit in ${file}`, () => {
      const options = JSON.parse(
        readFileSync(resolve(srcRoot, "..", file), "utf8"),
      ).compilerOptions;

      expect(options.strict).toBe(strict);
      expect(options.skipLibCheck).toBe(skipLibCheck);
    });
  }
});
