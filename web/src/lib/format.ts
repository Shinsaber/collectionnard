/** Taille lisible dans la langue donnée : « 512 o », « 1,4 Mo » (fr), « 1.4 MB » (en). */
export function formatBytes(bytes: number, lang: string): string {
  const units = ["byte", "kilobyte", "megabyte", "gigabyte"] as const;
  let value = bytes;
  let i = 0;
  while (value >= 1024 && i < units.length - 1) {
    value /= 1024;
    i++;
  }
  return new Intl.NumberFormat(lang, {
    style: "unit",
    unit: units[i],
    unitDisplay: "short",
    maximumFractionDigits: i === 0 ? 0 : 1,
  }).format(value);
}
