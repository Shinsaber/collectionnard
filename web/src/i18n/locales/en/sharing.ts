// "sharing" catalog (English): visibility, sharing, public pages, directory (P4-2).
import type fr from "../fr/sharing";

const en: typeof fr = {
  visibility: {
    label: "Visibility",
    private: "Private",
    members: "Members",
    public: "Public",
    privateHelp: "Only me.",
    membersHelp: "Any signed-in account of this instance.",
    publicHelp: "Everyone, even without an account.",
  },
  profile: {
    title: "Profile visibility",
    help: "Who sees my profile (display name, bio, avatar) and the list of my visible collections.",
    view: "View my public profile",
  },
  share: {
    button: "Share",
    title: "Share “{name}”",
    visibilityHelp:
      "A visible collection shows on my profile when the profile is visible too.",
    openPublic: "See the page as others see it",
    saved: "Sharing saved.",
    hidden: {
      title: "Fields hidden from others",
      help: "Removed by the server from everything others see (pages, links, statistics), administrators included.",
      byType: "Collection type setting",
      custom: "Setting of this collection",
      reset: "Back to the type setting",
      none: "This collection type has no field.",
    },
    links: {
      title: "Share links",
      help: "A secret link opens the collection read-only, even when private, until revoked.",
      create: "Create a link",
      none: "No active link.",
      created: "Created on {date}",
      revoke: "Revoke the link of {date}",
      revokeConfirm:
        "Revoke this link? Those who received it will no longer open the collection.",
      linkTitle: "Share link",
    },
  },
  public: {
    by: "by {name}",
    notFound: "This page does not exist, or is not visible.",
    signIn: "Sign in",
    mySpace: "My space",
    collections: "Collections",
    noCollections: "No visible collection.",
    memberSince: "Member since {date}",
    shared: "Collection shared by link, read-only",
    readonly: "Read-only",
  },
  members: {
    title: "Members",
    help: "Profiles open to members or public.",
    none: "No visible profile yet.",
  },
};

export default en;
