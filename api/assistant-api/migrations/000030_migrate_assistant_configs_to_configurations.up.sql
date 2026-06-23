INSERT INTO public.assistant_configurations (
    id,
    status,
    enabled,
    created_by,
    updated_by,
    created_date,
    updated_date,
    project_id,
    organization_id,
    assistant_id,
    configuration_type,
    provider
)
SELECT
    id,
    status,
    enabled,
    created_by,
    updated_by,
    created_date,
    updated_date,
    project_id,
    organization_id,
    assistant_id,
    configuration_type,
    provider
FROM (
    SELECT
        aa.id,
        aa.status,
        true AS enabled,
        aa.created_by,
        aa.updated_by,
        aa.created_date,
        aa.updated_date,
        aa.project_id,
        aa.organization_id,
        aa.assistant_id,
        'authentication'::character varying AS configuration_type,
        COALESCE(NULLIF(btrim(aa.provider), ''), 'http')::character varying AS provider
    FROM public.assistant_authentications aa

    UNION ALL

    SELECT
        aw.id,
        aw.status,
        true AS enabled,
        aw.created_by,
        aw.updated_by,
        aw.created_date,
        aw.updated_date,
        aw.project_id,
        aw.organization_id,
        aw.assistant_id,
        'webhook'::character varying AS configuration_type,
        COALESCE(NULLIF(btrim(aw.provider), ''), 'http')::character varying AS provider
    FROM public.assistant_webhooks aw

    UNION ALL

    SELECT
        aa.id,
        aa.status,
        true AS enabled,
        aa.created_by,
        aa.updated_by,
        aa.created_date,
        aa.updated_date,
        aa.project_id,
        aa.organization_id,
        aa.assistant_id,
        'analysis'::character varying AS configuration_type,
        COALESCE(NULLIF(btrim(aa.provider), ''), 'endpoint')::character varying AS provider
    FROM public.assistant_analyses aa

    UNION ALL

    SELECT
        atp.id,
        'ACTIVE'::character varying AS status,
        atp.enabled,
        NULL::bigint AS created_by,
        NULL::bigint AS updated_by,
        atp.created_date::timestamp without time zone AS created_date,
        atp.updated_date::timestamp without time zone AS updated_date,
        atp.project_id,
        atp.organization_id,
        atp.assistant_id,
        'telemetry'::character varying AS configuration_type,
        atp.provider_type::character varying AS provider
    FROM public.assistant_telemetry_providers atp
) source
ON CONFLICT (id) DO NOTHING;

CREATE SEQUENCE IF NOT EXISTS public.assistant_configuration_options_migration_id_seq;

WITH seed AS (
    SELECT MAX(id) AS max_id
    FROM public.assistant_configuration_options
)
SELECT setval(
    'public.assistant_configuration_options_migration_id_seq',
    COALESCE((SELECT max_id FROM seed), 1),
    COALESCE((SELECT max_id FROM seed), 0) > 0
);

INSERT INTO public.assistant_configuration_options (
    id,
    status,
    created_by,
    updated_by,
    created_date,
    updated_date,
    key,
    value,
    assistant_configuration_id
)
SELECT
    nextval('public.assistant_configuration_options_migration_id_seq'),
    aa.status,
    aa.created_by,
    aa.updated_by,
    aa.created_date,
    aa.updated_date,
    opt.key,
    opt.value,
    aa.id
FROM public.assistant_authentications aa
JOIN public.assistant_configurations ac
    ON ac.id = aa.id
    AND ac.configuration_type = 'authentication'
CROSS JOIN LATERAL (
    VALUES
        ('fail_behavior'::character varying, COALESCE(aa.fail_behavior, 'block')::text),
        ('timeout_ms'::character varying, COALESCE(aa.timeout_ms, 5000)::text)
) opt(key, value)
ON CONFLICT ON CONSTRAINT uk_assistant_configuration_options DO NOTHING;

INSERT INTO public.assistant_configuration_options (
    id,
    status,
    created_by,
    updated_by,
    created_date,
    updated_date,
    key,
    value,
    assistant_configuration_id
)
SELECT
    nextval('public.assistant_configuration_options_migration_id_seq'),
    aao.status,
    aao.created_by,
    aao.updated_by,
    aao.created_date,
    aao.updated_date,
    aao.key,
    aao.value,
    aao.assistant_authentication_id
FROM public.assistant_authentication_options aao
JOIN public.assistant_configurations ac
    ON ac.id = aao.assistant_authentication_id
    AND ac.configuration_type = 'authentication'
ON CONFLICT ON CONSTRAINT uk_assistant_configuration_options DO NOTHING;

INSERT INTO public.assistant_configuration_options (
    id,
    status,
    created_by,
    updated_by,
    created_date,
    updated_date,
    key,
    value,
    assistant_configuration_id
)
SELECT
    nextval('public.assistant_configuration_options_migration_id_seq'),
    aw.status,
    aw.created_by,
    aw.updated_by,
    aw.created_date,
    aw.updated_date,
    opt.key,
    opt.value,
    aw.id
FROM public.assistant_webhooks aw
JOIN public.assistant_configurations ac
    ON ac.id = aw.id
    AND ac.configuration_type = 'webhook'
CROSS JOIN LATERAL (
    VALUES
        ('description'::character varying, COALESCE(aw.description, '')::text),
        ('assistant_events'::character varying, COALESCE(aw.assistant_events, '[]')::text),
        ('execution_priority'::character varying, COALESCE(aw.execution_priority, 0)::text)
) opt(key, value)
ON CONFLICT ON CONSTRAINT uk_assistant_configuration_options DO NOTHING;

INSERT INTO public.assistant_configuration_options (
    id,
    status,
    created_by,
    updated_by,
    created_date,
    updated_date,
    key,
    value,
    assistant_configuration_id
)
SELECT
    nextval('public.assistant_configuration_options_migration_id_seq'),
    awo.status,
    awo.created_by,
    awo.updated_by,
    awo.created_date,
    awo.updated_date,
    awo.key,
    awo.value,
    awo.assistant_webhook_id
FROM public.assistant_webhook_options awo
JOIN public.assistant_configurations ac
    ON ac.id = awo.assistant_webhook_id
    AND ac.configuration_type = 'webhook'
ON CONFLICT ON CONSTRAINT uk_assistant_configuration_options DO NOTHING;

INSERT INTO public.assistant_configuration_options (
    id,
    status,
    created_by,
    updated_by,
    created_date,
    updated_date,
    key,
    value,
    assistant_configuration_id
)
SELECT
    nextval('public.assistant_configuration_options_migration_id_seq'),
    aa.status,
    aa.created_by,
    aa.updated_by,
    aa.created_date,
    aa.updated_date,
    opt.key,
    opt.value,
    aa.id
FROM public.assistant_analyses aa
JOIN public.assistant_configurations ac
    ON ac.id = aa.id
    AND ac.configuration_type = 'analysis'
CROSS JOIN LATERAL (
    VALUES
        ('name'::character varying, COALESCE(aa.name, '')::text),
        ('description'::character varying, COALESCE(aa.description, '')::text),
        ('execution_priority'::character varying, COALESCE(aa.execution_priority, 0)::text)
) opt(key, value)
ON CONFLICT ON CONSTRAINT uk_assistant_configuration_options DO NOTHING;

INSERT INTO public.assistant_configuration_options (
    id,
    status,
    created_by,
    updated_by,
    created_date,
    updated_date,
    key,
    value,
    assistant_configuration_id
)
SELECT
    nextval('public.assistant_configuration_options_migration_id_seq'),
    aao.status,
    aao.created_by,
    aao.updated_by,
    aao.created_date,
    aao.updated_date,
    aao.key,
    aao.value,
    aao.assistant_analysis_id
FROM public.assistant_analysis_options aao
JOIN public.assistant_configurations ac
    ON ac.id = aao.assistant_analysis_id
    AND ac.configuration_type = 'analysis'
ON CONFLICT ON CONSTRAINT uk_assistant_configuration_options DO NOTHING;

INSERT INTO public.assistant_configuration_options (
    id,
    status,
    created_by,
    updated_by,
    created_date,
    updated_date,
    key,
    value,
    assistant_configuration_id
)
SELECT
    nextval('public.assistant_configuration_options_migration_id_seq'),
    CASE WHEN upper(atpo.status) = 'ACTIVE' THEN 'ACTIVE' ELSE atpo.status END,
    atpo.created_by,
    atpo.updated_by,
    atpo.created_date::timestamp without time zone,
    atpo.updated_date::timestamp without time zone,
    atpo.key,
    atpo.value,
    atpo.assistant_telemetry_provider_id
FROM public.assistant_telemetry_provider_options atpo
JOIN public.assistant_configurations ac
    ON ac.id = atpo.assistant_telemetry_provider_id
    AND ac.configuration_type = 'telemetry'
ON CONFLICT ON CONSTRAINT uk_assistant_configuration_options DO NOTHING;

DROP SEQUENCE IF EXISTS public.assistant_configuration_options_migration_id_seq;

DROP TABLE IF EXISTS public.assistant_authentication_options;
DROP TABLE IF EXISTS public.assistant_authentications;

DROP TABLE IF EXISTS public.assistant_webhook_options;
DROP TABLE IF EXISTS public.assistant_webhooks;

DROP TABLE IF EXISTS public.assistant_analysis_options;
DROP TABLE IF EXISTS public.assistant_analyses;

DROP TABLE IF EXISTS public.assistant_telemetry_provider_options;
DROP TABLE IF EXISTS public.assistant_telemetry_providers;
