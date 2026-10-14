// Régénère tests/fixtures/seed-schemas.json depuis les seeds SQL (méta-types b…, c…, d…).
// Usage (depuis web/) : node scripts/seed-fixtures.mjs tests/fixtures/seed-schemas.json ../db/seed.sql ../db/seed_bd_manga.sql
import { readFileSync, writeFileSync } from "node:fs";
import process from "node:process";
const [, , outPath, ...seeds] = process.argv;
const re =
  /\('([bcd]0{7}-[0-9-]+)', '([A-Za-z_]+)', '((?:[^']|'')*)', '(\{(?:[^']|'')*\})'\)/g;
const out = [];
for (const seed of seeds) {
  const sql = readFileSync(seed, "utf8");
  for (const m of sql.matchAll(re)) {
    out.push({
      id: m[1],
      name: m[2],
      schema: JSON.parse(m[4].replace(/''/g, "'")),
    });
  }
}
writeFileSync(outPath, JSON.stringify(out, null, 2) + "\n");
console.log(out.length, "schemas");
