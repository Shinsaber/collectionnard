-- P4-3c — Notifications dans l'application (social.md).
--
-- Une ligne par événement pour un compte : décision sur une de ses propositions, nouveau
-- message dans une discussion. `payload` porte les références (proposition, titre). Plus tard,
-- un résumé quotidien des non-lues pourra partir par e-mail (SMTP).
CREATE TABLE notifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind        VARCHAR(40) NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    read_at     TIMESTAMPTZ
);
CREATE INDEX notifications_user_idx ON notifications (user_id, created_at DESC);
CREATE INDEX notifications_unread_idx ON notifications (user_id) WHERE read_at IS NULL;

---- create above / drop below ----

DROP TABLE IF EXISTS notifications;
