-- Migration 005 : ordre d'affichage des types d'item (petit = en premier, puis par nom).
--
-- Les types principaux (Series, Manga…) passent avant les types volumineux (Episode, Volume…)
-- dans « Répartition par type », les sélecteurs et les listes. Défaut : 100.
-- Les valeurs ci-dessous ne touchent que les types des seeds (UUID fixes) ; les mêmes UPDATE
-- figurent à la fin de seed.sql et seed_bd_manga.sql pour une base neuve.
ALTER TABLE item_types ADD COLUMN display_order INTEGER NOT NULL DEFAULT 100;

UPDATE item_types t SET display_order = v.o
FROM (VALUES
    ('b0000000-0000-0000-0000-000000000001'::uuid, 10),  -- Series
    ('b0000000-0000-0000-0000-000000000012'::uuid, 20),  -- Manga
    ('b0000000-0000-0000-0000-000000000013'::uuid, 30),  -- Comic
    ('b0000000-0000-0000-0000-000000000010'::uuid, 40),  -- Universe
    ('b0000000-0000-0000-0000-000000000011'::uuid, 50),  -- Saga
    ('b0000000-0000-0000-0000-000000000005'::uuid, 60),  -- DVD
    ('b0000000-0000-0000-0000-000000000014'::uuid, 70),  -- Edition
    ('b0000000-0000-0000-0000-000000000003'::uuid, 80),  -- Season
    ('b0000000-0000-0000-0000-000000000002'::uuid, 90),  -- Broadcast
    ('b0000000-0000-0000-0000-000000000015'::uuid, 110), -- Volume
    ('b0000000-0000-0000-0000-000000000004'::uuid, 120), -- Episode
    ('b0000000-0000-0000-0000-000000000016'::uuid, 130), -- Person
    ('b0000000-0000-0000-0000-000000000017'::uuid, 140)  -- Publisher
) AS v(id, o)
WHERE t.id = v.id;

---- create above / drop below ----

ALTER TABLE item_types DROP COLUMN display_order;
