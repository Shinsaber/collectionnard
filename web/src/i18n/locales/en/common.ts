import type fr from "../fr/common";

const en: typeof fr = {
  appName: "Collectionnard",
  actions: "Actions",
  add: "Add",
  back: "Back",
  cancel: "Cancel",
  close: "Close",
  confirm: "Confirm",
  confirmation: "Confirmation",
  create: "Create",
  delete: "Delete",
  description: "Description",
  edit: "Edit",
  loading: "Loading…",
  name: "Name",
  no: "No",
  none: "None",
  save: "Save",
  search: "Search",
  type: "Type",
  yes: "Yes",
  error: "Error",
  created: "Created",
  updated: "Updated",
  deleted: "Deleted",
  enlarge: "Enlarge: {name}",
  // Barcode scanner (lot M2)
  scan: {
    button: "Scan a barcode",
    title: "Scan a barcode",
    preview: "Camera preview",
    hint: "Aim at the barcode (ISBN, EAN…): it is read automatically. If it stays unreadable, take a photo of it.",
    detected: "Code read: {code}",
    torchOn: "Turn the light on",
    torchOff: "Turn the light off",
    fromPhoto: "From a photo",
    noCamera: "No camera available: use “From a photo”.",
    denied:
      "Camera access denied: allow it in the browser, or use “From a photo”.",
    unsupported: "Barcode reading is not available in this browser.",
    failed: "Barcode reading failed",
    nothingFound: "No readable barcode in this image.",
  },
};

export default en;
