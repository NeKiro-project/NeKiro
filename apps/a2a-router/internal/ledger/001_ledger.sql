CREATE SCHEMA IF NOT EXISTS ledger;

CREATE TABLE ledger.invocations (
    invocation_id varchar(128) COLLATE "C" PRIMARY KEY,
    root_task_id varchar(128) COLLATE "C" NOT NULL,
    parent_invocation_id varchar(128) COLLATE "C",
    trace_id varchar(128) COLLATE "C" NOT NULL,
    caller_type varchar(16) COLLATE "C" NOT NULL,
    caller_id varchar(128) COLLATE "C" NOT NULL,
    workspace_id varchar(128) COLLATE "C" NOT NULL,
    target_agent_id varchar(128) COLLATE "C" NOT NULL,
    agent_card_version varchar(128) COLLATE "C" NOT NULL,
    capability varchar(128) COLLATE "C" NOT NULL,
    status varchar(16) COLLATE "C" NOT NULL,
    latency_ms bigint,
    error_code varchar(64) COLLATE "C",
    created_at timestamptz(6) NOT NULL,
    updated_at timestamptz(6) NOT NULL,
    CONSTRAINT invocations_parent_fk FOREIGN KEY (parent_invocation_id)
        REFERENCES ledger.invocations (invocation_id),
    CONSTRAINT invocations_identifier_format CHECK (
        invocation_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
        AND root_task_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
        AND (parent_invocation_id IS NULL OR parent_invocation_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$')
        AND caller_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
        AND workspace_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
        AND target_agent_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
        AND capability ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
    ),
    CONSTRAINT invocations_trace_format CHECK (trace_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    CONSTRAINT invocations_caller_type CHECK (caller_type IN ('user', 'service', 'agent')),
    CONSTRAINT invocations_status CHECK (status IN ('pending', 'routing', 'running', 'succeeded', 'failed', 'canceled', 'timed_out')),
    CONSTRAINT invocations_latency_nonnegative CHECK (latency_ms IS NULL OR latency_ms >= 0),
    CONSTRAINT invocations_error_code CHECK (error_code IS NULL OR error_code IN (
        'VALIDATION_ERROR','UNAUTHENTICATED','FORBIDDEN','NOT_FOUND','CONFLICT','NOT_ACCEPTABLE','PAYLOAD_TOO_LARGE',
        'AGENT_NOT_INSTALLED','INSTALLATION_DISABLED','AGENT_DISABLED','CAPABILITY_NOT_ALLOWED','ROUTE_NOT_FOUND',
        'AGENT_AUTH_UNSUPPORTED','AGENT_RESPONSE_TOO_LARGE','A2A_PROTOCOL_ERROR','AGENT_UNAVAILABLE',
        'AGENT_EXECUTION_FAILED','DEPENDENCY_ERROR','TIMEOUT','CANCELED','INTERNAL_ERROR'
    )),
    CONSTRAINT invocations_timestamp_order CHECK (created_at <= updated_at),
    CONSTRAINT invocations_terminal_shape CHECK (
        (status IN ('pending', 'routing', 'running') AND latency_ms IS NULL AND error_code IS NULL)
        OR (status = 'succeeded' AND latency_ms IS NOT NULL AND error_code IS NULL)
        OR (status IN ('failed', 'canceled', 'timed_out') AND latency_ms IS NOT NULL AND error_code IS NOT NULL)
    )
);

CREATE INDEX invocations_trace_order_idx
    ON ledger.invocations (workspace_id, trace_id, created_at, invocation_id);
CREATE INDEX invocations_root_order_idx
    ON ledger.invocations (workspace_id, root_task_id, created_at, invocation_id);
CREATE INDEX invocations_parent_order_idx
    ON ledger.invocations (workspace_id, parent_invocation_id, created_at, invocation_id);

CREATE TABLE ledger.invocation_events (
    event_id varchar(128) COLLATE "C" PRIMARY KEY,
    invocation_id varchar(128) COLLATE "C" NOT NULL,
    sequence bigint NOT NULL,
    occurred_at timestamptz(6) NOT NULL,
    event_type varchar(16) COLLATE "C" NOT NULL,
    status varchar(16) COLLATE "C" NOT NULL,
    root_task_id varchar(128) COLLATE "C" NOT NULL,
    parent_invocation_id varchar(128) COLLATE "C",
    trace_id varchar(128) COLLATE "C" NOT NULL,
    caller_type varchar(16) COLLATE "C" NOT NULL,
    caller_id varchar(128) COLLATE "C" NOT NULL,
    workspace_id varchar(128) COLLATE "C" NOT NULL,
    target_agent_id varchar(128) COLLATE "C" NOT NULL,
    agent_card_version varchar(128) COLLATE "C" NOT NULL,
    capability varchar(128) COLLATE "C" NOT NULL,
    chunk_index bigint,
    chunk_bytes bigint,
    latency_ms bigint,
    error_code varchar(64) COLLATE "C",
    CONSTRAINT invocation_events_invocation_fk FOREIGN KEY (invocation_id)
        REFERENCES ledger.invocations (invocation_id),
    CONSTRAINT invocation_events_sequence_unique UNIQUE (invocation_id, sequence),
    CONSTRAINT invocation_events_sequence_nonnegative CHECK (sequence >= 0),
    CONSTRAINT invocation_events_identifier_format CHECK (
        event_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
        AND invocation_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
        AND root_task_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
        AND (parent_invocation_id IS NULL OR parent_invocation_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$')
        AND caller_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
        AND workspace_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
        AND target_agent_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
        AND capability ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
    ),
    CONSTRAINT invocation_events_trace_format CHECK (trace_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    CONSTRAINT invocation_events_caller_type CHECK (caller_type IN ('user', 'service', 'agent')),
    CONSTRAINT invocation_events_counter_nonnegative CHECK (
        (chunk_index IS NULL OR chunk_index >= 0)
        AND (chunk_bytes IS NULL OR chunk_bytes >= 0)
        AND (latency_ms IS NULL OR latency_ms >= 0)
    ),
    CONSTRAINT invocation_events_type_status CHECK (
        (event_type = 'created' AND status = 'pending')
        OR (event_type = 'routing' AND status = 'routing')
        OR (event_type IN ('started', 'stream') AND status = 'running')
        OR (event_type = status AND event_type IN ('succeeded', 'failed', 'canceled', 'timed_out'))
    ),
    CONSTRAINT invocation_events_field_shape CHECK (
        (event_type IN ('created', 'routing', 'started') AND chunk_index IS NULL AND chunk_bytes IS NULL AND latency_ms IS NULL AND error_code IS NULL)
        OR (event_type = 'stream' AND chunk_index IS NOT NULL AND chunk_bytes IS NOT NULL AND latency_ms IS NULL AND error_code IS NULL)
        OR (event_type = 'succeeded' AND chunk_index IS NULL AND chunk_bytes IS NULL AND latency_ms IS NOT NULL AND error_code IS NULL)
        OR (event_type IN ('failed', 'canceled', 'timed_out') AND chunk_index IS NULL AND chunk_bytes IS NULL AND latency_ms IS NOT NULL AND error_code IS NOT NULL)
    ),
    CONSTRAINT invocation_events_terminal_error CHECK (
        (event_type = 'canceled' AND error_code = 'CANCELED')
        OR (event_type = 'timed_out' AND error_code = 'TIMEOUT')
        OR (event_type NOT IN ('canceled', 'timed_out'))
    ),
    CONSTRAINT invocation_events_error_code CHECK (error_code IS NULL OR error_code IN (
        'VALIDATION_ERROR','UNAUTHENTICATED','FORBIDDEN','NOT_FOUND','CONFLICT','NOT_ACCEPTABLE','PAYLOAD_TOO_LARGE',
        'AGENT_NOT_INSTALLED','INSTALLATION_DISABLED','AGENT_DISABLED','CAPABILITY_NOT_ALLOWED','ROUTE_NOT_FOUND',
        'AGENT_AUTH_UNSUPPORTED','AGENT_RESPONSE_TOO_LARGE','A2A_PROTOCOL_ERROR','AGENT_UNAVAILABLE',
        'AGENT_EXECUTION_FAILED','DEPENDENCY_ERROR','TIMEOUT','CANCELED','INTERNAL_ERROR'
    )
);

CREATE FUNCTION ledger.reject_invocation_event_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'invocation events are immutable' USING ERRCODE = '55000';
END;
$$;

CREATE TRIGGER invocation_events_immutable
BEFORE UPDATE OR DELETE ON ledger.invocation_events
FOR EACH ROW EXECUTE FUNCTION ledger.reject_invocation_event_mutation();

---- create above / drop below ----

DROP TABLE ledger.invocation_events;
DROP FUNCTION ledger.reject_invocation_event_mutation();
DROP TABLE ledger.invocations;
