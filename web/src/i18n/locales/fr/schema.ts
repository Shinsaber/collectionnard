// Formulaire dynamique (DynamicForm, ImageGalleryField) et éditeur de schéma (SchemaBuilder…).
export default {
  example: "ex. {v}",
  types: {
    string: "Texte",
    integer: "Entier",
    number: "Nombre",
    boolean: "Booléen",
    array: "Tableau",
    i18nText: "Texte multilingue",
    imageGallery: "Galerie d'images",
    links: "Liens",
    object: "Objet",
  },
  formats: {
    uri: "URI / URL",
    date: "Date",
    dateTime: "Date et heure",
    email: "E-mail",
  },
  form: {
    choose: "Choisir…",
    noAttributes: "Aucun attribut défini pour ce type.",
    languages: "Langue de saisie",
    linkAdd: "Ajouter un lien",
    linkLabel: "Libellé du lien {n}",
    linkLabelPlaceholder: "Libellé (facultatif)",
    linkUrl: "Adresse du lien {n}",
    linkUrlPlaceholder: "Adresse (https://…)",
    linkRemove: "Retirer le lien {n}",
  },
  gallery: {
    setDefault: "Définir comme image par défaut",
    setDefaultN: "Définir l'image {n} de {category} comme image par défaut",
    removeN: "Supprimer l'image {n} de {category}",
    file: "Fichier image pour {category}",
    uploadFailed: "Envoi de l'image impossible",
    refused: "Image refusée",
    warned: "Image ajoutée, mais : {details}",
    ratioOff:
      "ratio {actual} au lieu de {expected} (écart {deviation} %, toléré {tolerance} %)",
    tooSmall: "{width} × {height} px, minimum {min} px",
    // Photo et recadrage (lot M1)
    photo: "Photo",
    photoLabel: "Prendre une photo pour {category}",
    cropTitle: "Recadrer – {category}",
    cropHint:
      "Cadre au ratio {ratio} : déplacez-le ou tirez ses poignées ; faites glisser l'image pour la déplacer, pincez (ou molette) pour zoomer.",
    cropHintFree:
      "Cadre libre : déplacez-le ou tirez ses poignées ; faites glisser l'image pour la déplacer, pincez (ou molette) pour zoomer.",
    cropOffer: "Image hors format ({details}) : recadrez-la pour la garder.",
    rotateLeft: "Pivoter d'un quart de tour vers la gauche",
    rotateRight: "Pivoter d'un quart de tour vers la droite",
    fitImage: "Ajuster",
    asIs: "Sans recadrage",
    applyCrop: "Recadrer",
    cropFailed: "Recadrage impossible",
    // Redressement de la perspective (lot M4)
    straightenTitle: "Redresser – {category}",
    straightenStep: "Redresser",
    straightenHint:
      "Placez les quatre coins sur ceux de la couverture (détectés d'office quand le fond contraste), puis redressez. Flèches au clavier : 1 % par appui, 10 % avec Maj.",
    detectEdges: "Détecter les bords",
    wholeImage: "Image entière",
    skipStraighten: "Passer",
    straighten: "Redresser",
    noEdges: "Bords non détectés : placez les coins à la main, ou passez.",
    roughEdges: "Bords détectés approximativement : vérifiez les coins.",
    corner: "Coin {n}",
    straightenFailed: "Redressement impossible",
    // Prise de vue dans l'application (retours v0.4.0)
    cameraTitle: "Photo – {category}",
    cameraHint:
      "Cadrez la couverture, puis déclenchez : elle sera ensuite redressée et recadrée.",
    shutter: "Prendre la photo",
    nativeCamera: "Appareil photo du téléphone",
    cameraNone:
      "Aucune caméra accessible ici : utilisez l'appareil photo du téléphone.",
    cameraDenied:
      "Accès à la caméra refusé : autorisez-le dans le navigateur, ou utilisez l'appareil photo du téléphone.",
    captureFailed: "Prise de vue impossible",
    convertOffer:
      "Format {type} non accepté tel quel : l'image sera convertie en JPEG à l'envoi.",
  },
  builder: {
    empty:
      "Aucun champ défini. Cliquez sur « Ajouter un champ » pour commencer.",
    fields: "Champs",
    addField: "Ajouter un champ",
    showJson: "Voir le JSON",
    hideJson: "Masquer le JSON",
    generatedJson: "JSON Schema généré",
  },
  root: {
    heading: "Affichage",
    title: "Nom affiché ({lang})",
    titlePlaceholder: "ex. Série",
    description: "Description ({lang})",
    titleHelp:
      "Nom et description du type tels qu'affichés dans l'interface ; sans traduction, l'identifiant technique est utilisé.",
    xLabel: "Libellé des instances",
    xLabelPlaceholder:
      "ex. {'{'}title{'}'} ou {'{'}number{'}'}. {'{'}title{'}'}",
    xLabelHelp:
      "Gabarit avec les clés des champs entre accolades ; sans gabarit, le premier champ title, name ou label est utilisé.",
    completeness: "Relation de composition",
    completenessHelp:
      "La cible fait partie de la source (tome d'une édition, épisode d'une saison…) : les statistiques comptent les cibles possédées sur le total (« 30 tomes sur 42 »).",
  },
  field: {
    moveUp: "Monter le champ",
    moveDown: "Descendre le champ",
    key: "Clé du champ",
    keyPlaceholder: "clé (ex. title)",
    type: "Type du champ",
    required: "Champ requis",
    requiredShort: "Req.",
    private:
      "Privé : caché à tout autre que le propriétaire de la collection (prix payé…), même publique",
    privateShort: "Privé",
    identifier:
      "Identifiant (ISBN, EAN, QID…) : un autre item portant la même valeur est signalé comme doublon possible",
    identifierShort: "Identifiant",
    remove: "Supprimer le champ",
    addAfter: "Ajouter un champ après",
  },
  constraints: {
    label: "Libellé {lang}",
    labelPlaceholder: "ex. Titre",
    descriptionPlaceholder: "Description du champ",
    minLength: "Min. caractères",
    maxLength: "Max. caractères",
    pattern: "Motif (regex)",
    format: "Format",
    enum: "Valeurs (enum, séparées par des virgules)",
    default: "Valeur par défaut",
    minimum: "Min",
    maximum: "Max",
    exclusiveMinimum: "Min exclusif",
    exclusiveMaximum: "Max exclusif",
    multipleOf: "Multiple de",
    linksHelp:
      "Liste de liens : une adresse (http ou https) et un libellé facultatif par ligne ; sans libellé, le nom du site est affiché.",
    i18nTextHelp:
      "Génère un objet avec une chaîne par langue ({langs}) : les utilisateurs saisiront une valeur par langue.",
    imageCategories: "Catégories d'images",
    categoryName: "Nom (ex. poster)",
    categoryNameN: "Nom de la catégorie {n}",
    categoryRatio: "Ratio (ex. 2/3)",
    categoryRatioN: "Ratio de la catégorie {n}",
    categoryRemoveN: "Supprimer la catégorie {n}",
    addCategory: "Ajouter une catégorie",
    categoryChecks: "Contrôles à l'envoi de la catégorie {n}",
    tolerance: "Écart de ratio toléré (%)",
    minWidth: "Largeur min. (px)",
    minHeight: "Hauteur min. (px)",
    strict: "Refuser l'image hors contrôles",
    checksHelp:
      "Facultatif. Sans écart toléré, le ratio n'est pas contrôlé. Sans « Refuser », une image hors contrôles est ajoutée avec un avertissement.",
    imageGalleryHelp:
      "Génère un objet avec un tableau d'URLs par catégorie et un sélecteur d'image par défaut. Le ratio est au format CSS (ex. 2/3, 16/9, 3/1).",
    itemsType: "Type des éléments",
    uniqueItems: "Éléments uniques",
    minItems: "Min. éléments",
    maxItems: "Max. éléments",
  },
  enumLabels: {
    caption: "Libellés affichés des valeurs",
    value: "Valeur",
    inputLabel: "Libellé de « {value} » en {lang}",
  },
};
