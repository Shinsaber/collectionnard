// Régénère tests/fixtures/seed-schemas.json depuis les paquets de domaine (méta-types b…, c…, d…).
// Usage (depuis web/) :
//   node scripts/seed-fixtures.mjs tests/fixtures/seed-schemas.json ../server/internal/domains/packages/{collections,tv,books}.json
//   npx prettier --write tests/fixtures/seed-schemas.json
import { readFileSync, writeFileSync } from "node:fs";
import process from "node:process";
const [, , outPath, ...packages] = process.argv;
const out = [];
for (const file of packages) {
  const pkg = JSON.parse(readFileSync(file, "utf8"));
  for (const t of [
    ...pkg.item_types,
    ...pkg.relation_types,
    ...pkg.collection_types,
  ]) {
    out.push({ id: t.id, name: t.name, schema: t.attributes_schema });
  }
}
// Ordre stable : par identifiant (b… types d'item, c… relations, d… collections).
out.sort((a, b) => a.id.localeCompare(b.id));
writeFileSync(outPath, JSON.stringify(out, null, 2) + "\n");
console.log(out.length, "schemas");
