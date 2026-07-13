ALTER TABLE catalog.agent_versions
    ADD COLUMN card_name text,
    ADD COLUMN card_description text;

UPDATE catalog.agent_versions
SET card_name = card->>'name',
    card_description = card->>'description';

ALTER TABLE catalog.agent_versions
    ALTER COLUMN card_name SET NOT NULL,
    ALTER COLUMN card_description SET NOT NULL,
    ALTER COLUMN card TYPE text USING card::text;

---- create above / drop below ----

ALTER TABLE catalog.agent_versions
    ALTER COLUMN card TYPE jsonb USING card::jsonb,
    DROP COLUMN card_description,
    DROP COLUMN card_name;
