// "friends" catalog (English): friends, requests and invitations (P7-2).
import type fr from "../fr/friends";

const en: typeof fr = {
  title: "Friends",
  help: "Your friends see what you keep for them: collections, profile and reviews set to “Friends”.",
  invite: {
    label: "Invite a member",
    placeholder: "Username",
    help: "A member whose profile you can see (directory, reviews, comments). For a private profile, send them a friend link.",
    button: "Invite as a friend",
    notFound:
      "No member whose profile you can see has this name. For a private profile, send them a friend link.",
    submit: "Invite",
    sent: "Request sent to {name}.",
    accepted: "You and {name} are now friends.",
  },
  received: "Requests received",
  sent: "Requests sent",
  list: "My friends",
  none: "No friends yet: invite a member, or send a friend link.",
  since: "Friends since {date}",
  requested: "Request of {date}",
  accept: "Accept",
  refuse: "Refuse",
  cancel: "Cancel",
  remove: "Remove",
  removeConfirm:
    "Remove {name} from your friends? What you keep for your friends will no longer be visible to them.",
  removed: "Removed from your friends.",
  refused: "Request refused.",
  cancelled: "Request cancelled.",
  profile: "View profile",
  links: {
    title: "Friend links",
    help: "A secret link, valid 7 days and usable once: whoever opens it becomes your friend by accepting. Hand it out yourself, to one person.",
    create: "Create a link",
    linkTitle: "Friend link",
    share: "Share",
    shareText: "Be my friend on Collectionnard",
    none: "No valid link.",
    expires: "Expires on {date}",
    revoke: "Revoke",
    revoked: "Link revoked.",
  },
  link: {
    title: "Invitation",
    invites: "{name} would like to be friends.",
    help: "Your friends see what you keep for them, and you see what they keep for you.",
    accept: "Accept",
    later: "Later",
    invalid: "This link does not exist, has expired or was already used.",
    own: "This is your own link: hand it to the person you invite.",
  },
};

export default en;
