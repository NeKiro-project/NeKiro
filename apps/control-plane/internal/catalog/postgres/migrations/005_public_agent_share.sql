ALTER TABLE catalog.agent_identities
    ADD COLUMN public_agent_id varchar(36) COLLATE "C";

UPDATE catalog.agent_identities
SET public_agent_id = 'agt_' || md5(agent_id)
WHERE public_agent_id IS NULL;

ALTER TABLE catalog.agent_identities
    ALTER COLUMN public_agent_id SET NOT NULL,
    ADD CONSTRAINT agent_identities_public_agent_id_format CHECK (public_agent_id ~ '^agt_[0-9a-f]{32}$');

CREATE UNIQUE INDEX agent_identities_public_agent_id_idx
    ON catalog.agent_identities (public_agent_id);

CREATE FUNCTION catalog.reject_public_agent_id_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.public_agent_id <> OLD.public_agent_id THEN
        RAISE EXCEPTION 'public Agent ID is immutable' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER agent_identities_public_agent_id_immutable
BEFORE UPDATE ON catalog.agent_identities
FOR EACH ROW EXECUTE FUNCTION catalog.reject_public_agent_id_mutation();

---- create above / drop below ----

DROP TRIGGER agent_identities_public_agent_id_immutable ON catalog.agent_identities;
DROP FUNCTION catalog.reject_public_agent_id_mutation();
DROP INDEX catalog.agent_identities_public_agent_id_idx;
ALTER TABLE catalog.agent_identities
    DROP CONSTRAINT agent_identities_public_agent_id_format,
    DROP COLUMN public_agent_id;
