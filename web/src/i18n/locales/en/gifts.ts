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
  handOver: "I handed it over",
  handOverConfirm:
    "Mark this gift as handed over? The owner will be told: the surprise is over.",
  handedOver: "Handover noted: the owner will confirm they received it.",
  given: "Handed over, awaiting confirmation",
  received: "Received on {date}",
  receivedAll: "Received included",
  remove: "Remove",
  removeConfirm: "Remove this gift from your list?",
  removed: "Removed.",
  givenMark: "given",
  leftList: "Left their list",
  owner: {
    flag: "{n} given by {names} | {n} given by {names}",
    receive: "Received",
    decline: "Not received",
    declineConfirm:
      "Mark this gift as not received? {name}'s reservation will be cancelled and they will be told.",
    declined: "Reservation cancelled.",
    receiveTitle: "Received: “{label}”",
    receiveHelp:
      "{n} copy leaves your list. Where to put it? | {n} copies leave your list. Where to put them?",
    moveTo: "In one of my collections",
    removeOnly: "Just remove it (I already added it)",
    noTarget: "None of your other collections accepts this item.",
    attributes: "The new lot",
    submit: "Confirm",
    received: "Gift received.",
  },
  leftHelp:
    "This gift is no longer on their list: mark it given, or cancel the reservation.",
};

export default en;
