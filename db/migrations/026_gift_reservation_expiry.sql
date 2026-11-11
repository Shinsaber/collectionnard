-- P7-4 — Durée limitée d'une réservation de cadeau (docs/phases/exemplaires.md).
--
-- Une réservation non offerte vaut jusqu'à expires_at (inclus) : au-delà, elle ne compte plus dans le lot et sort de la
-- vue des amis, mais l'offrant la garde dans « À offrir » (« Expirée ») pour la prolonger ou l'annuler. Le serveur
-- applique l'échéance à chaque lecture ; expiry_reminded_at : rappel « bientôt expirée » envoyé pour l'échéance en
-- cours (remis à zéro quand elle change). Les réservations existantes reçoivent 60 jours.
ALTER TABLE gift_reservations
    ADD COLUMN expires_at DATE NOT NULL DEFAULT (current_date + 60),
    ADD COLUMN expiry_reminded_at TIMESTAMPTZ;
CREATE INDEX gift_reservations_expiring ON gift_reservations (expires_at) WHERE status = 'reserved';

---- create above / drop below ----

DROP INDEX IF EXISTS gift_reservations_expiring;
ALTER TABLE gift_reservations DROP COLUMN IF EXISTS expiry_reminded_at, DROP COLUMN IF EXISTS expires_at;
