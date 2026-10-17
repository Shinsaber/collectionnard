import type fr from "../fr/account";

// "account" catalogue: accounts, invitations, profile (Phase 4, P4-0).
const en: typeof fr = {
  email: "Email address",
  displayName: "Display name",
  bio: "About",
  role: "Role",
  roles: { user: "Member", admin: "Admin" },
  avatarAlt: "Avatar of {name}",
  register: {
    subtitle: "Create your account",
    invalid:
      "This invitation link is no longer valid (already used, expired or withdrawn). Ask for a new one.",
    usernameHelp:
      "2 to 100 characters: letters, digits, « _ », « . » or « - ».",
    submit: "Create my account",
  },
  reset: {
    subtitle: "Choose a new password",
    submit: "Save the password",
    done: "Password changed: you can sign in.",
  },
  password: {
    title: "Password",
    current: "Current password",
    new: "New password",
    confirm: "Confirm the password",
    help: "At least {n} characters.",
    mismatch: "The two entries differ.",
    submit: "Change the password",
    saved: "Password changed; your other sessions are closed.",
  },
  profile: {
    title: "My profile",
    subtitle: "Account {username} · {email}",
    section: "Profile",
    avatarChoose: "Choose an avatar",
    avatarRemove: "Remove the avatar",
    saved: "Profile saved",
  },
  link: {
    once: "This link is shown only once: copy it and pass it on yourself.",
    expires: "Valid until {date}, for a single use.",
    copy: "Copy the link",
    copied: "Copied",
  },
  invitations: {
    title: "Invitations",
    help: "A single-use link lets someone create an account on this instance.",
    days: "Validity (days)",
    note: "Memo",
    notePlaceholder: "For whom? (optional)",
    create: "Create an invitation",
    none: "No invitation",
    by: "by {name}",
    status: "Status",
    pending: "Pending, until {date}",
    used: "Used by {name}",
    expired: "Expired",
    withdraw: "Withdraw the invitation",
    withdrawConfirm: "Withdraw this invitation? Its link will stop working.",
    linkTitle: "Invitation link",
  },
  users: {
    title: "Users",
    help: "Accounts of the instance: role, activation, password reset (a link to pass on, no email).",
    account: "Account",
    roleOf: "Role of {name}",
    active: "Active",
    activeOf: "Account of {name} active",
    resetLink: "Reset link",
    resetLinkTitle: "Password reset for {name}",
    disableConfirm:
      "Disable the account of {name}? Their sessions are closed and they can no longer sign in.",
    saved: "Account of {name} updated",
  },
};

export default en;
