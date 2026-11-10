// "gifts" catalog (English): gifts reserved on a friend's wish list (P7-4).
import type fr from "../fr/gifts";

const en: typeof fr = {
  tab: "To give",
  none: "No gift reserved.",
  reserve: "Reserve",
  mine: "My reservation",
  reserveTitle: "Reserve “{label}”",
  help: "The owner of the list will not know: only their friends see reservations, to avoid duplicates.",
  quantity: "Copies",
  left: "none left to reserve | 1 left to reserve | {n} left to reserve",
  note: "Note (for you only)",
  until: "Reserved until",
  untilHelp:
    "Past this date, the reservation no longer holds for your friends: you extend or cancel it.",
  untilOn: "until {date}",
  expired: "Expired",
  expiredHelp:
    "Expired on {date}: your friends no longer see it. Extend it, or cancel it.",
  extended: "Due date saved.",
  submit: "Reserve",
  save: "Save",
  cancel: "Cancel the reservation",
  cancelConfirm: "Cancel your reservation?",
  reserved: "Reserved.",
  saved: "Reservation saved.",
  cancelled: "Reservation cancelled.",
  flag: "{names} · {n} of {total}",
  you: "you",
  full: "All reserved",
  for: "For {name}",
  given: "Given",
  markGiven: "It's given",
  markReserved: "Not given yet",
  leftList: "Left their list",
  leftHelp:
    "This gift is no longer on their list: mark it given, or cancel the reservation.",
};

export default en;
