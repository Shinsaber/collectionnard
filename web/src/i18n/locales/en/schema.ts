import type fr from "../fr/schema";

const en: typeof fr = {
  example: "e.g. {v}",
  types: {
    string: "Text",
    integer: "Integer",
    number: "Number",
    boolean: "Boolean",
    array: "Array",
    i18nText: "Multilingual text",
    imageGallery: "Image gallery",
    links: "Links",
    object: "Object",
  },
  formats: {
    uri: "URI / URL",
    date: "Date",
    dateTime: "Date and time",
    email: "Email",
  },
  form: {
    choose: "Choose…",
    noAttributes: "No attribute defined for this type.",
    languages: "Input language",
    linkAdd: "Add a link",
    linkLabel: "Label of link {n}",
    linkLabelPlaceholder: "Label (optional)",
    linkUrl: "Address of link {n}",
    linkUrlPlaceholder: "Address (https://…)",
    linkRemove: "Remove link {n}",
  },
  gallery: {
    setDefault: "Set as default image",
    setDefaultN: "Set image {n} of {category} as default image",
    removeN: "Remove image {n} of {category}",
    file: "Image file for {category}",
    uploadFailed: "Image upload failed",
    refused: "Image refused",
    warned: "Image added, but: {details}",
    ratioOff:
      "ratio {actual} instead of {expected} ({deviation} % off, {tolerance} % allowed)",
    tooSmall: "{width} × {height} px, minimum {min} px",
    // Photo and cropping (lot M1)
    photo: "Photo",
    photoLabel: "Take a photo for {category}",
    cropTitle: "Crop – {category}",
    cropHint:
      "Frame at ratio {ratio}: move it or drag its handles; drag the image to move it, pinch (or scroll) to zoom.",
    cropHintFree:
      "Free frame: move it or drag its handles; drag the image to move it, pinch (or scroll) to zoom.",
    cropOffer: "Image out of format ({details}): crop it to keep it.",
    rotateLeft: "Rotate a quarter turn left",
    rotateRight: "Rotate a quarter turn right",
    fitImage: "Fit",
    asIs: "Without cropping",
    applyCrop: "Crop",
    cropFailed: "Cropping failed",
    // Perspective straightening (lot M4)
    straightenTitle: "Straighten – {category}",
    straightenStep: "Straighten",
    straightenHint:
      "Put the four corners on those of the cover (detected automatically when the background contrasts), then straighten. Arrow keys: 1% per press, 10% with Shift.",
    detectEdges: "Detect edges",
    wholeImage: "Whole image",
    skipStraighten: "Skip",
    straighten: "Straighten",
    noEdges: "Edges not detected: place the corners by hand, or skip.",
    roughEdges: "Edges detected roughly: check the corners.",
    corner: "Corner {n}",
    straightenFailed: "Straightening failed",
    // In-app camera (v0.4.0 feedback)
    cameraTitle: "Photo – {category}",
    cameraHint:
      "Frame the cover, then shoot: it will be straightened and cropped next.",
    shutter: "Take the photo",
    nativeCamera: "Phone camera app",
    cameraNone: "No camera reachable here: use the phone camera app.",
    cameraDenied:
      "Camera access denied: allow it in the browser, or use the phone camera app.",
    captureFailed: "Could not take the photo",
    convertOffer:
      "Format {type} is not accepted as is: the image will be converted to JPEG on upload.",
  },
  builder: {
    empty: "No field defined. Click “Add a field” to start.",
    fields: "Fields",
    addField: "Add a field",
    showJson: "Show JSON",
    hideJson: "Hide JSON",
    generatedJson: "Generated JSON Schema",
  },
  root: {
    heading: "Display",
    title: "Display name ({lang})",
    titlePlaceholder: "e.g. Series",
    description: "Description ({lang})",
    titleHelp:
      "Type name and description as displayed in the interface; without a translation, the technical identifier is used.",
    xLabel: "Instance label",
    xLabelPlaceholder:
      "e.g. {'{'}title{'}'} or {'{'}number{'}'}. {'{'}title{'}'}",
    xLabelHelp:
      "Template with field keys between braces; without a template, the first title, name or label field is used.",
    completeness: "Composition relation",
    completenessHelp:
      'The target is part of the source (volume of an edition, episode of a season…): statistics count the owned targets out of the total ("30 volumes of 42").',
  },
  field: {
    moveUp: "Move field up",
    moveDown: "Move field down",
    key: "Field key",
    keyPlaceholder: "key (e.g. title)",
    type: "Field type",
    required: "Required field",
    requiredShort: "Req.",
    private:
      "Private: hidden from anyone but the collection owner (price paid…), even when public",
    privateShort: "Private",
    identifier:
      "Identifier (ISBN, EAN, QID…): another item with the same value is reported as a possible duplicate",
    identifierShort: "Identifier",
    remove: "Remove field",
    addAfter: "Add a field after",
  },
  constraints: {
    label: "Label {lang}",
    labelPlaceholder: "e.g. Title",
    descriptionPlaceholder: "Field description",
    minLength: "Min. characters",
    maxLength: "Max. characters",
    pattern: "Pattern (regex)",
    format: "Format",
    enum: "Values (enum, comma-separated)",
    default: "Default value",
    minimum: "Min",
    maximum: "Max",
    exclusiveMinimum: "Exclusive min",
    exclusiveMaximum: "Exclusive max",
    multipleOf: "Multiple of",
    linksHelp:
      "List of links: one address (http or https) and an optional label per row; without a label, the site name is shown.",
    i18nTextHelp:
      "Generates an object with one string per language ({langs}): users will enter one value per language.",
    imageCategories: "Image categories",
    categoryName: "Name (e.g. poster)",
    categoryNameN: "Name of category {n}",
    categoryRatio: "Ratio (e.g. 2/3)",
    categoryRatioN: "Ratio of category {n}",
    categoryRemoveN: "Remove category {n}",
    addCategory: "Add a category",
    categoryChecks: "Upload checks of category {n}",
    tolerance: "Allowed ratio deviation (%)",
    minWidth: "Min. width (px)",
    minHeight: "Min. height (px)",
    strict: "Refuse images failing the checks",
    checksHelp:
      "Optional. Without an allowed deviation the ratio is not checked. Without “Refuse”, an image failing the checks is added with a warning.",
    imageGalleryHelp:
      "Generates an object with an array of URLs per category and a default image selector. The ratio uses the CSS format (e.g. 2/3, 16/9, 3/1).",
    itemsType: "Item type",
    uniqueItems: "Unique items",
    minItems: "Min. items",
    maxItems: "Max. items",
  },
  enumLabels: {
    caption: "Displayed labels of the values",
    value: "Value",
    inputLabel: "Label of “{value}” in {lang}",
  },
};

export default en;
