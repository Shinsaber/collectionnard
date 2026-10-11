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
  },
  gallery: {
    setDefault: "Set as default image",
    setDefaultN: "Set image {n} of {category} as default image",
    removeN: "Remove image {n} of {category}",
    file: "Image file for {category}",
    default: "Default image:",
    uploadFailed: "Image upload failed",
  },
  builder: {
    empty: "No field defined. Click “Add a field” to start.",
    addField: "Add a field",
    showJson: "Show JSON",
    hideJson: "Hide JSON",
    generatedJson: "Generated JSON Schema",
  },
  root: {
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
  },
  field: {
    moveUp: "Move field up",
    moveDown: "Move field down",
    key: "Field key",
    keyPlaceholder: "key (e.g. title)",
    type: "Field type",
    required: "Required field",
    requiredShort: "Req.",
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
    i18nTextHelp:
      "Generates an object with one string per language ({langs}): users will enter one value per language.",
    imageCategories: "Image categories",
    categoryName: "Name (e.g. poster)",
    categoryNameN: "Name of category {n}",
    categoryRatio: "Ratio (e.g. 2/3)",
    categoryRatioN: "Ratio of category {n}",
    categoryRemoveN: "Remove category {n}",
    addCategory: "Add a category",
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
