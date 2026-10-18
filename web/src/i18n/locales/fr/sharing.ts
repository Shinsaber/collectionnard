// Catalogue « sharing » (français) : visibilité, partage, pages publiques, annuaire (P4-2).
export default {
  visibility: {
    label: "Visibilité",
    private: "Privée",
    members: "Membres",
    public: "Publique",
    privateHelp: "Moi seul.",
    membersHelp: "Tout compte connecté de cette instance.",
    publicHelp: "Tout le monde, même sans compte.",
  },
  profile: {
    title: "Visibilité du profil",
    help: "Qui voit mon profil (nom affiché, présentation, avatar) et la liste de mes collections visibles.",
    view: "Voir mon profil public",
  },
  share: {
    button: "Partager",
    title: "Partager « {name} »",
    visibilityHelp:
      "Une collection visible apparaît sur mon profil s'il est lui-même visible.",
    openPublic: "Voir la page telle que les autres la voient",
    saved: "Partage enregistré.",
    hidden: {
      title: "Champs cachés aux autres",
      help: "Retirés par le serveur de tout ce que voient les autres (pages, liens, statistiques), administrateurs compris.",
      byType: "Réglage du type de collection",
      custom: "Choix propre à cette collection",
      reset: "Revenir au réglage du type",
      none: "Ce type de collection n'a aucun champ.",
    },
    links: {
      title: "Liens de partage",
      help: "Un lien secret ouvre la collection en lecture seule, même privée, jusqu'à sa révocation.",
      create: "Créer un lien",
      none: "Aucun lien actif.",
      created: "Créé le {date}",
      revoke: "Révoquer le lien du {date}",
      revokeConfirm:
        "Révoquer ce lien ? Ceux qui l'ont reçu ne pourront plus ouvrir la collection.",
      linkTitle: "Lien de partage",
    },
  },
  public: {
    by: "par {name}",
    notFound: "Cette page n'existe pas, ou n'est pas visible.",
    signIn: "Se connecter",
    mySpace: "Mon espace",
    collections: "Collections",
    noCollections: "Aucune collection visible.",
    memberSince: "Membre depuis {date}",
    shared: "Collection partagée par lien, en lecture seule",
    readonly: "Lecture seule",
  },
  members: {
    title: "Membres",
    help: "Profils ouverts aux membres ou publics.",
    none: "Aucun profil visible pour l'instant.",
  },
};
