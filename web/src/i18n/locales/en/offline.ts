import type fr from "../fr/offline";

const en: typeof fr = {
  update: {
    title: "New version",
    message: "A new version of Collectionnard is available.",
    reload: "Reload",
    later: "Later",
  },
  title: "Offline",
  subtitle: "Copy of my collections, available without network",
  dataOf: "Data from {date}",
  never: "No copy on this device yet",
  noCopy: "No offline copy: open the app once with network to build it.",
  offlineNow: "No network",
  onlineNow: "Online",
  sync: "Synchronize",
  syncing: "Synchronizing…",
  banner: "No network: only the copy of your collections is available.",
  bannerLink: "Open the copy",
  retry: "Retry",
  search: "Title, number, ISBN…",
  searchLabel: "Search my collections",
  allCollections: "All",
  results: "No item | {n} item | {n} items",
  more: "{n} more results: refine the search.",
  empty: "No item matches.",
  coversTitle: "Offline covers",
  coversHelp:
    "Light thumbnails of the items in your collections, kept on the device up to the chosen limit; beyond it, the least recently viewed are removed.",
  coversEnabled: "Keep covers",
  coversLimit: "Limit",
  coversUsage: "{count} cover(s), {size} of {limit}",
  storageUsage: "Space used by the app: {size}",
  clear: "Clear",
  clearConfirm:
    "Delete the offline copy and covers from this device? They will be rebuilt at the next synchronization.",
  cleared: "Offline data deleted",
};

export default en;
