CREATE TABLE public.assistant_authentications (
    id bigint PRIMARY KEY,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    assistant_id bigint NOT NULL,
    fail_behavior character varying(20) NOT NULL DEFAULT 'block',
    timeout_ms bigint NOT NULL DEFAULT 5000,
    project_id bigint NOT NULL,
    organization_id bigint NOT NULL,
    provider character varying(50) DEFAULT 'http'::character varying NOT NULL
);

CREATE INDEX idx_assistant_authentications_assistant_id
    ON public.assistant_authentications USING btree (assistant_id);

CREATE INDEX idx_assistant_authentications_assistant_id_status
    ON public.assistant_authentications USING btree (assistant_id, status);

CREATE INDEX idx_assistant_authentications_project_id
    ON public.assistant_authentications USING btree (project_id);

CREATE INDEX idx_assistant_authentications_organization_id
    ON public.assistant_authentications USING btree (organization_id);

CREATE INDEX idx_assistant_authentications_assistant_org_project
    ON public.assistant_authentications USING btree (assistant_id, organization_id, project_id);

CREATE TABLE public.assistant_authentication_options (
    id bigint PRIMARY KEY,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    assistant_authentication_id bigint NOT NULL
);

ALTER TABLE ONLY public.assistant_authentication_options
    ADD CONSTRAINT uk_assistant_authentication_options UNIQUE (key, assistant_authentication_id);

CREATE INDEX idx_assistant_authentication_options_auth_id
    ON public.assistant_authentication_options USING btree (assistant_authentication_id);

CREATE TABLE public.assistant_webhooks (
    id bigint PRIMARY KEY,
    assistant_id bigint NOT NULL,
    assistant_events text NOT NULL,
    description text,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    execution_priority bigint DEFAULT '-1'::integer,
    project_id bigint NOT NULL,
    organization_id bigint NOT NULL,
    provider character varying(50) DEFAULT 'http'::character varying NOT NULL
);

CREATE INDEX idx_assistant_webhooks_assistant_id
    ON public.assistant_webhooks USING btree (assistant_id);

CREATE INDEX idx_assistant_webhooks_status
    ON public.assistant_webhooks USING btree (status);

CREATE INDEX idx_assistant_webhooks_project_id
    ON public.assistant_webhooks USING btree (project_id);

CREATE INDEX idx_assistant_webhooks_organization_id
    ON public.assistant_webhooks USING btree (organization_id);

CREATE INDEX idx_assistant_webhooks_assistant_org_project
    ON public.assistant_webhooks USING btree (assistant_id, organization_id, project_id);

CREATE TABLE public.assistant_webhook_options (
    id bigint PRIMARY KEY,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    assistant_webhook_id bigint NOT NULL
);

ALTER TABLE ONLY public.assistant_webhook_options
    ADD CONSTRAINT uk_assistant_webhook_option UNIQUE (key, assistant_webhook_id);

CREATE INDEX idx_assistant_webhook_options_assistant_webhook_id
    ON public.assistant_webhook_options USING btree (assistant_webhook_id);

CREATE TABLE public.assistant_analyses (
    id bigint PRIMARY KEY,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    assistant_id bigint NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    name character varying(200) NOT NULL,
    description text NOT NULL,
    execution_priority bigint NOT NULL,
    project_id bigint NOT NULL,
    organization_id bigint NOT NULL,
    provider character varying(50) DEFAULT 'endpoint'::character varying NOT NULL
);

CREATE INDEX assistant_analyses_assistant_id_idx
    ON public.assistant_analyses USING btree (assistant_id);

CREATE INDEX assistant_analyses_created_date_idx
    ON public.assistant_analyses USING btree (created_date);

CREATE INDEX idx_assistant_analyses_project_id
    ON public.assistant_analyses USING btree (project_id);

CREATE INDEX idx_assistant_analyses_organization_id
    ON public.assistant_analyses USING btree (organization_id);

CREATE INDEX idx_assistant_analyses_assistant_org_project
    ON public.assistant_analyses USING btree (assistant_id, organization_id, project_id);

CREATE TABLE public.assistant_analysis_options (
    id bigint PRIMARY KEY,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    assistant_analysis_id bigint NOT NULL
);

ALTER TABLE ONLY public.assistant_analysis_options
    ADD CONSTRAINT uk_assistant_analysis_option UNIQUE (key, assistant_analysis_id);

CREATE INDEX idx_assistant_analysis_options_assistant_analysis_id
    ON public.assistant_analysis_options USING btree (assistant_analysis_id);

CREATE TABLE public.assistant_telemetry_providers (
    id bigint NOT NULL PRIMARY KEY,
    project_id bigint NOT NULL,
    organization_id bigint NOT NULL,
    assistant_id bigint NOT NULL,
    provider_type character varying(50) NOT NULL,
    enabled boolean NOT NULL DEFAULT true,
    created_date timestamp with time zone NOT NULL DEFAULT now(),
    updated_date timestamp with time zone DEFAULT NULL
);

CREATE INDEX idx_atp_assistant_id
    ON public.assistant_telemetry_providers USING btree (assistant_id);

CREATE INDEX idx_atp_project_id
    ON public.assistant_telemetry_providers USING btree (project_id);

CREATE TABLE public.assistant_telemetry_provider_options (
    id bigint NOT NULL PRIMARY KEY,
    assistant_telemetry_provider_id bigint NOT NULL REFERENCES public.assistant_telemetry_providers(id) ON DELETE CASCADE,
    key character varying(200) NOT NULL,
    value character varying(1000) NOT NULL DEFAULT '',
    status character varying(50) NOT NULL DEFAULT 'ACTIVE',
    created_by bigint,
    updated_by bigint,
    created_date timestamp with time zone NOT NULL DEFAULT now(),
    updated_date timestamp with time zone DEFAULT NULL
);

CREATE INDEX idx_atpo_provider_id
    ON public.assistant_telemetry_provider_options USING btree (assistant_telemetry_provider_id);

INSERT INTO public.assistant_authentications (
    id,
    status,
    created_by,
    updated_by,
    created_date,
    updated_date,
    assistant_id,
    fail_behavior,
    timeout_ms,
    project_id,
    organization_id,
    provider
)
SELECT
    ac.id,
    ac.status,
    COALESCE(ac.created_by, 0),
    ac.updated_by,
    ac.created_date,
    ac.updated_date,
    ac.assistant_id,
    COALESCE(fail_behavior.value, 'block'),
    CASE
        WHEN timeout_ms.value ~ '^[0-9]+$' THEN timeout_ms.value::bigint
        ELSE 5000
    END,
    ac.project_id,
    ac.organization_id,
    ac.provider
FROM public.assistant_configurations ac
LEFT JOIN public.assistant_configuration_options fail_behavior
    ON fail_behavior.assistant_configuration_id = ac.id
    AND fail_behavior.key = 'fail_behavior'
    AND fail_behavior.status = 'ACTIVE'
LEFT JOIN public.assistant_configuration_options timeout_ms
    ON timeout_ms.assistant_configuration_id = ac.id
    AND timeout_ms.key = 'timeout_ms'
    AND timeout_ms.status = 'ACTIVE'
WHERE ac.configuration_type = 'authentication'
ON CONFLICT (id) DO NOTHING;

INSERT INTO public.assistant_authentication_options (
    id,
    status,
    created_by,
    updated_by,
    created_date,
    updated_date,
    key,
    value,
    assistant_authentication_id
)
SELECT
    aco.id,
    aco.status,
    COALESCE(aco.created_by, 0),
    aco.updated_by,
    aco.created_date,
    aco.updated_date,
    aco.key,
    aco.value,
    aco.assistant_configuration_id
FROM public.assistant_configuration_options aco
JOIN public.assistant_configurations ac
    ON ac.id = aco.assistant_configuration_id
    AND ac.configuration_type = 'authentication'
WHERE aco.key NOT IN ('fail_behavior', 'timeout_ms')
ON CONFLICT ON CONSTRAINT uk_assistant_authentication_options DO NOTHING;

INSERT INTO public.assistant_webhooks (
    id,
    assistant_id,
    assistant_events,
    description,
    status,
    created_by,
    updated_by,
    created_date,
    updated_date,
    execution_priority,
    project_id,
    organization_id,
    provider
)
SELECT
    ac.id,
    ac.assistant_id,
    COALESCE(assistant_events.value, '[]'),
    description.value,
    ac.status,
    COALESCE(ac.created_by, 0),
    ac.updated_by,
    ac.created_date,
    ac.updated_date,
    CASE
        WHEN execution_priority.value ~ '^[0-9]+$' THEN execution_priority.value::bigint
        ELSE -1
    END,
    ac.project_id,
    ac.organization_id,
    ac.provider
FROM public.assistant_configurations ac
LEFT JOIN public.assistant_configuration_options assistant_events
    ON assistant_events.assistant_configuration_id = ac.id
    AND assistant_events.key = 'assistant_events'
    AND assistant_events.status = 'ACTIVE'
LEFT JOIN public.assistant_configuration_options description
    ON description.assistant_configuration_id = ac.id
    AND description.key = 'description'
    AND description.status = 'ACTIVE'
LEFT JOIN public.assistant_configuration_options execution_priority
    ON execution_priority.assistant_configuration_id = ac.id
    AND execution_priority.key = 'execution_priority'
    AND execution_priority.status = 'ACTIVE'
WHERE ac.configuration_type = 'webhook'
ON CONFLICT (id) DO NOTHING;

INSERT INTO public.assistant_webhook_options (
    id,
    status,
    created_by,
    updated_by,
    created_date,
    updated_date,
    key,
    value,
    assistant_webhook_id
)
SELECT
    aco.id,
    aco.status,
    COALESCE(aco.created_by, 0),
    aco.updated_by,
    aco.created_date,
    aco.updated_date,
    aco.key,
    aco.value,
    aco.assistant_configuration_id
FROM public.assistant_configuration_options aco
JOIN public.assistant_configurations ac
    ON ac.id = aco.assistant_configuration_id
    AND ac.configuration_type = 'webhook'
WHERE aco.key NOT IN ('description', 'assistant_events', 'execution_priority')
ON CONFLICT ON CONSTRAINT uk_assistant_webhook_option DO NOTHING;

INSERT INTO public.assistant_analyses (
    id,
    status,
    assistant_id,
    created_by,
    updated_by,
    created_date,
    updated_date,
    name,
    description,
    execution_priority,
    project_id,
    organization_id,
    provider
)
SELECT
    ac.id,
    ac.status,
    ac.assistant_id,
    COALESCE(ac.created_by, 0),
    ac.updated_by,
    ac.created_date,
    ac.updated_date,
    COALESCE(name.value, ''),
    COALESCE(description.value, ''),
    CASE
        WHEN execution_priority.value ~ '^[0-9]+$' THEN execution_priority.value::bigint
        ELSE 0
    END,
    ac.project_id,
    ac.organization_id,
    ac.provider
FROM public.assistant_configurations ac
LEFT JOIN public.assistant_configuration_options name
    ON name.assistant_configuration_id = ac.id
    AND name.key = 'name'
    AND name.status = 'ACTIVE'
LEFT JOIN public.assistant_configuration_options description
    ON description.assistant_configuration_id = ac.id
    AND description.key = 'description'
    AND description.status = 'ACTIVE'
LEFT JOIN public.assistant_configuration_options execution_priority
    ON execution_priority.assistant_configuration_id = ac.id
    AND execution_priority.key = 'execution_priority'
    AND execution_priority.status = 'ACTIVE'
WHERE ac.configuration_type = 'analysis'
ON CONFLICT (id) DO NOTHING;

INSERT INTO public.assistant_analysis_options (
    id,
    status,
    created_by,
    updated_by,
    created_date,
    updated_date,
    key,
    value,
    assistant_analysis_id
)
SELECT
    aco.id,
    aco.status,
    COALESCE(aco.created_by, 0),
    aco.updated_by,
    aco.created_date,
    aco.updated_date,
    aco.key,
    aco.value,
    aco.assistant_configuration_id
FROM public.assistant_configuration_options aco
JOIN public.assistant_configurations ac
    ON ac.id = aco.assistant_configuration_id
    AND ac.configuration_type = 'analysis'
WHERE aco.key NOT IN ('name', 'description', 'execution_priority')
ON CONFLICT ON CONSTRAINT uk_assistant_analysis_option DO NOTHING;

INSERT INTO public.assistant_telemetry_providers (
    id,
    project_id,
    organization_id,
    assistant_id,
    provider_type,
    enabled,
    created_date,
    updated_date
)
SELECT
    ac.id,
    ac.project_id,
    ac.organization_id,
    ac.assistant_id,
    ac.provider,
    ac.enabled,
    ac.created_date,
    ac.updated_date
FROM public.assistant_configurations ac
WHERE ac.configuration_type = 'telemetry'
ON CONFLICT (id) DO NOTHING;

INSERT INTO public.assistant_telemetry_provider_options (
    id,
    assistant_telemetry_provider_id,
    key,
    value,
    status,
    created_by,
    updated_by,
    created_date,
    updated_date
)
SELECT
    aco.id,
    aco.assistant_configuration_id,
    aco.key,
    LEFT(aco.value, 1000),
    aco.status,
    aco.created_by,
    aco.updated_by,
    aco.created_date,
    aco.updated_date
FROM public.assistant_configuration_options aco
JOIN public.assistant_configurations ac
    ON ac.id = aco.assistant_configuration_id
    AND ac.configuration_type = 'telemetry'
ON CONFLICT (id) DO NOTHING;
