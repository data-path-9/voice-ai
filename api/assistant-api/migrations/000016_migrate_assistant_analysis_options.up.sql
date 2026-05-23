CREATE SEQUENCE IF NOT EXISTS public.assistant_analysis_options_backfill_id_seq;

WITH seed AS (
    SELECT MAX(id) AS max_id
    FROM public.assistant_analysis_options
)
SELECT setval(
    'public.assistant_analysis_options_backfill_id_seq',
    COALESCE((SELECT max_id FROM seed), 1),
    COALESCE((SELECT max_id FROM seed), 0) > 0
);

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
    nextval('public.assistant_analysis_options_backfill_id_seq'),
    aa.status,
    aa.created_by,
    aa.updated_by,
    aa.created_date,
    aa.updated_date,
    'endpoint_id',
    aa.endpoint_id::text,
    aa.id
FROM public.assistant_analyses aa
ON CONFLICT ON CONSTRAINT uk_assistant_analysis_option DO NOTHING;

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
    nextval('public.assistant_analysis_options_backfill_id_seq'),
    aa.status,
    aa.created_by,
    aa.updated_by,
    aa.created_date,
    aa.updated_date,
    'endpoint_version',
    aa.endpoint_version::text,
    aa.id
FROM public.assistant_analyses aa
ON CONFLICT ON CONSTRAINT uk_assistant_analysis_option DO NOTHING;

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
    nextval('public.assistant_analysis_options_backfill_id_seq'),
    aa.status,
    aa.created_by,
    aa.updated_by,
    aa.created_date,
    aa.updated_date,
    'endpoint_parameters',
    aa.endpoint_parameters,
    aa.id
FROM public.assistant_analyses aa
ON CONFLICT ON CONSTRAINT uk_assistant_analysis_option DO NOTHING;

DROP SEQUENCE IF EXISTS public.assistant_analysis_options_backfill_id_seq;

DROP INDEX IF EXISTS public.assistant_analyses_endpoint_id_idx;
DROP INDEX IF EXISTS public.assistant_analyses_endpoint_version_idx;

ALTER TABLE public.assistant_analyses
    DROP COLUMN IF EXISTS endpoint_id,
    DROP COLUMN IF EXISTS endpoint_version,
    DROP COLUMN IF EXISTS endpoint_parameters;
