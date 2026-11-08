// Catalogue français — langue de référence : sa forme (`Messages`) est imposée aux autres
// langues par le compilateur. Un fichier par espace de noms (`t("items.title")`…).
import account from "./account";
import admin from "./admin";
import articles from "./articles";
import auth from "./auth";
import collections from "./collections";
import common from "./common";
import dashboard from "./dashboard";
import errors from "./errors";
import friends from "./friends";
import graph from "./graph";
import history from "./history";
import items from "./items";
import layout from "./layout";
import moderation from "./moderation";
import notifications from "./notifications";
import offline from "./offline";
import proposals from "./proposals";
import recommendations from "./recommendations";
import reviews from "./reviews";
import scan from "./scan";
import schema from "./schema";
import sharing from "./sharing";
import stats from "./stats";

const fr = {
  account,
  admin,
  articles,
  auth,
  collections,
  common,
  dashboard,
  errors,
  friends,
  graph,
  history,
  items,
  layout,
  moderation,
  notifications,
  offline,
  proposals,
  recommendations,
  reviews,
  scan,
  schema,
  sharing,
  stats,
};

export type Messages = typeof fr;

export default fr;
