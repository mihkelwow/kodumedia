import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import path from "node:path";

const scriptDirectory = path.dirname(fileURLToPath(import.meta.url));
const webDirectory = path.resolve(scriptDirectory, "..");
const lockPath = path.join(webDirectory, "package-lock.json");
const lock = JSON.parse(await readFile(lockPath, "utf8"));
const allowedHost = "registry.npmjs.org";
const errors = [];

for (const [packagePath, metadata] of Object.entries(lock.packages ?? {})) {
  const resolved = metadata?.resolved;
  if (typeof resolved !== "string" || resolved.length === 0) continue;

  if (resolved.startsWith("http://") || resolved.startsWith("https://")) {
    let host;
    try {
      host = new URL(resolved).host;
    } catch {
      errors.push(`${packagePath || "<root>"}: invalid resolved URL: ${resolved}`);
      continue;
    }
    if (host !== allowedHost) {
      errors.push(`${packagePath || "<root>"}: unexpected registry host ${host}`);
    }
  }
}

if (errors.length > 0) {
  console.error("package-lock.json must use only the public npm registry.");
  for (const error of errors) console.error(`- ${error}`);
  process.exit(1);
}

console.log(`Registry check passed: all package tarballs use https://${allowedHost}/`);
