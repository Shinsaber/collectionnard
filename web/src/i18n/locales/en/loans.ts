// "loans" catalog (English): loans and borrowings (P7-3).
import type fr from "../fr/loans";

const en: typeof fr = {
  title: "Loans and gifts",
  help: "What you lent, what your friends lent you, and the gifts you plan to give them.",
  tabs: { lent: "Lent", borrowed: "Borrowed" },
  status: { all: "All, returned included" },
  none: { lent: "No loan.", borrowed: "Nothing borrowed." },
  lend: "Lend…",
  loansOf: "Loans…",
  lendTitle: "Lend “{label}”",
  loansTitle: "Loans of “{label}”",
  borrower: "Borrower",
  friend: "A friend",
  name: "A name",
  friendPlaceholder: "Choose a friend",
  namePlaceholder: "Name of the person",
  noFriends:
    "No friends yet: lend to a name, or invite them from the Members page.",
  friendHelp: "Your friend gets a notification and sees this loan.",
  nameHelp: "Nobody but you sees this loan.",
  quantity: "Copies",
  available: "none available | 1 available | {n} available",
  lentAt: "Lent on",
  dueAt: "Due back",
  note: "Note (for you only)",
  submit: "Lend",
  created: "Loan recorded.",
  markReturned: "Returned today",
  returnedDone: "Loan closed.",
  delete: "Delete",
  deleteConfirm:
    "Delete this loan? A returned loan is kept as history: only delete a loan entered by mistake.",
  deleted: "Loan deleted.",
  dueSaved: "Due date saved.",
  flag: "{n} lent · {who} | {n} lent · {who}",
  overdue: "Overdue",
  dueOn: "due {date}",
  lentOn: "lent {date}",
  returnedOn: "returned {date}",
  by: "Lent by {name}",
  to: "Lent to {name}",
  active: "In progress",
  history: "History",
  noneLot: "No loan for this lot.",
};

export default en;
