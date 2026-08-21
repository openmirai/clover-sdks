#!/usr/bin/env node
/**
 * Bootstrap a disposable Clover API key + verified domain for SDK live E2E.
 *
 * Prerequisites: clover Compose up (`make up`), migrate applied, API + worker running.
 * Uses clover repo paths from CLOVER_ROOT (default: sibling ../clover).
 */
import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const CLOVER_ROOT =
  process.env.CLOVER_ROOT ||
  path.resolve(__dirname, "../../../../clover");
const API_URL = (process.env.CLOVER_API_URL || "http://127.0.0.1:8080").replace(/\/$/, "");
const PRIVATE_KEY = path.join(CLOVER_ROOT, ".local/dashboard-grant-private.pem");
const MINT = path.join(CLOVER_ROOT, "tests/mint-dashboard-grant.mjs");
const NONCE = `${Date.now()}-${process.pid}`;

function mint(subject, email, org = "", role = "") {
  const args = [
    MINT,
    PRIVATE_KEY,
    process.env.CLOVER_DASHBOARD_GRANT_ISSUER || "http://localhost:3000",
    process.env.CLOVER_DASHBOARD_GRANT_AUDIENCE || "clover-api",
    process.env.CLOVER_DASHBOARD_GRANT_KEY_ID || "local-es256-1",
    subject,
    email,
    org,
    role,
  ].filter((v) => v !== "");
  const result = spawnSync(process.execPath, args, { encoding: "utf8" });
  if (result.status !== 0) {
    throw new Error(result.stderr || result.stdout || "mint failed");
  }
  return result.stdout.trim();
}

async function request(method, apiPath, { token, body, idempotency } = {}) {
  const headers = {
    Accept: "application/json",
    Authorization: `Bearer ${token}`,
    "User-Agent": "clover-sdk-live-e2e/0.1.0",
  };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (idempotency) headers["Idempotency-Key"] = idempotency;
  const response = await fetch(`${API_URL}${apiPath}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await response.text();
  let json = {};
  try {
    json = text ? JSON.parse(text) : {};
  } catch {
    json = { raw: text };
  }
  return { status: response.status, json };
}

function unwrap(json) {
  if (json && typeof json.success === "boolean") return json.data ?? {};
  return json;
}

async function main() {
  if (!fs.existsSync(PRIVATE_KEY)) {
    throw new Error(`missing dashboard grant key at ${PRIVATE_KEY}; run make setup in clover`);
  }

  const bootstrapGrant = mint(`sdk-e2e-${NONCE}`, `sdk-e2e-${NONCE}@example.test`);
  const bootstrap = await request("POST", "/api/v1/dashboard/bootstrap", { token: bootstrapGrant });
  if (bootstrap.status !== 200) {
    throw new Error(`bootstrap failed: ${bootstrap.status} ${JSON.stringify(bootstrap.json)}`);
  }
  const bootstrapData = unwrap(bootstrap.json);
  const orgs = bootstrapData.organizations || bootstrap.json.organizations;
  const orgId = orgs?.[0]?.id;
  if (!orgId) throw new Error(`no organization in bootstrap: ${JSON.stringify(bootstrap.json)}`);

  const ownerGrant = mint(`sdk-e2e-${NONCE}`, `sdk-e2e-${NONCE}@example.test`, orgId, "owner");
  const keyRes = await request("POST", "/api/v1/api-keys", {
    token: ownerGrant,
    body: { name: `sdk-e2e-${NONCE}`, permission: "full_access", domain_id: null },
    idempotency: `sdk-key-${NONCE}`,
  });
  if (keyRes.status !== 201) {
    throw new Error(`create api key failed: ${keyRes.status} ${JSON.stringify(keyRes.json)}`);
  }
  const keyData = unwrap(keyRes.json);
  const apiKey = keyData.token;
  if (!apiKey) throw new Error(`missing token: ${JSON.stringify(keyRes.json)}`);

  const domainName = `sdk-${NONCE}.test`;
  const domainRes = await request("POST", "/api/v1/domains", {
    token: apiKey,
    body: {
      name: domainName,
      provider: "smtp",
      region: "local",
      receivingEnabled: false,
      tlsPolicy: "opportunistic",
    },
    idempotency: `sdk-domain-${NONCE}`,
  });
  if (domainRes.status !== 201) {
    throw new Error(`create domain failed: ${domainRes.status} ${JSON.stringify(domainRes.json)}`);
  }
  const domain = unwrap(domainRes.json);
  const domainId = domain.id;
  if (!domainId) throw new Error(`missing domain id: ${JSON.stringify(domainRes.json)}`);

  // Force-verify in local DB (DNS is not available for *.test).
  const compose = spawnSync(
    "docker",
    [
      "compose",
      "-f",
      path.join(CLOVER_ROOT, "docker-compose.yml"),
      "exec",
      "-T",
      "postgres",
      "psql",
      "-X",
      "-v",
      "ON_ERROR_STOP=1",
      "-U",
      process.env.POSTGRES_USER || "clover",
      "-d",
      process.env.POSTGRES_DB || "clover",
      "-c",
      `UPDATE clover.domains SET status='verified', sending_enabled=TRUE, last_verified_at=now(), updated_at=now() WHERE id='${domainId}';`,
    ],
    { encoding: "utf8", cwd: CLOVER_ROOT },
  );
  if (compose.status !== 0) {
    throw new Error(`domain force-verify failed: ${compose.stderr || compose.stdout}`);
  }

  const out = {
    apiUrl: API_URL,
    apiKey,
    domainId,
    domainName,
    from: `sender@${domainName}`,
    mailpitUrl: process.env.MAILPIT_URL || "http://127.0.0.1:8025",
    organizationId: orgId,
  };
  const outPath = process.env.CLOVER_E2E_FIXTURE || path.join(__dirname, "../.live-e2e-fixture.json");
  fs.writeFileSync(outPath, JSON.stringify(out, null, 2));
  process.stdout.write(`${outPath}\n`);
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
