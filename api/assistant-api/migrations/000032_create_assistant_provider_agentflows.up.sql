CREATE TABLE public.assistant_provider_agentflows (
    id bigint PRIMARY KEY,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    assistant_id bigint NOT NULL,
    description text,
    schema_version character varying(50) NOT NULL,
    definition jsonb NOT NULL
);

CREATE INDEX idx_assistant_provider_agentflows_assistant_id
    ON public.assistant_provider_agentflows USING btree (assistant_id);

CREATE INDEX idx_assistant_provider_agentflows_id_assistant_id
    ON public.assistant_provider_agentflows USING btree (id, assistant_id);
