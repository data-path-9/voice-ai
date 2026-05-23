ALTER TABLE public.assistant_analyses
    ADD COLUMN IF NOT EXISTS endpoint_id bigint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS endpoint_version character varying(200) NOT NULL DEFAULT 'latest',
    ADD COLUMN IF NOT EXISTS endpoint_parameters text NOT NULL DEFAULT '{}'::text;

WITH endpoint_ids AS (
    SELECT
        assistant_analysis_id,
        value
    FROM public.assistant_analysis_options
    WHERE key = 'endpoint_id'
),
endpoint_versions AS (
    SELECT
        assistant_analysis_id,
        value
    FROM public.assistant_analysis_options
    WHERE key = 'endpoint_version'
),
endpoint_parameters AS (
    SELECT
        assistant_analysis_id,
        value
    FROM public.assistant_analysis_options
    WHERE key = 'endpoint_parameters'
)
UPDATE public.assistant_analyses aa
SET
    endpoint_id = COALESCE((SELECT CAST(ei.value AS bigint) FROM endpoint_ids ei WHERE ei.assistant_analysis_id = aa.id LIMIT 1), 0),
    endpoint_version = COALESCE((SELECT ev.value FROM endpoint_versions ev WHERE ev.assistant_analysis_id = aa.id LIMIT 1), 'latest'),
    endpoint_parameters = COALESCE((SELECT ep.value FROM endpoint_parameters ep WHERE ep.assistant_analysis_id = aa.id LIMIT 1), '{}'::text);

CREATE INDEX IF NOT EXISTS assistant_analyses_endpoint_id_idx ON public.assistant_analyses (endpoint_id);
CREATE INDEX IF NOT EXISTS assistant_analyses_endpoint_version_idx ON public.assistant_analyses (endpoint_version);
