ALTER TABLE IF EXISTS public.assistant_webhook_logs
    RENAME TO assistant_http_logs;

ALTER TABLE IF EXISTS public.assistant_http_logs
    RENAME COLUMN webhook_id TO source_ref_id;

ALTER TABLE IF EXISTS public.assistant_http_logs
    RENAME COLUMN event TO source_event;

ALTER TABLE IF EXISTS public.assistant_http_logs
    ADD COLUMN IF NOT EXISTS source character varying(50) NOT NULL DEFAULT 'webhook'::character varying,
    ADD COLUMN IF NOT EXISTS context_id character varying(100),
    ADD COLUMN IF NOT EXISTS error_message text;

ALTER TABLE IF EXISTS public.assistant_http_logs
    ALTER COLUMN source_ref_id SET DEFAULT 0;

UPDATE public.assistant_http_logs
SET source_ref_id = 0
WHERE source_ref_id IS NULL;

ALTER TABLE IF EXISTS public.assistant_http_logs
    ALTER COLUMN source_ref_id SET NOT NULL;

DROP INDEX IF EXISTS public.idx_assistant_webhook_logs_assistant_id;
DROP INDEX IF EXISTS public.idx_assistant_webhook_logs_organization_id;
DROP INDEX IF EXISTS public.idx_assistant_webhook_logs_project_id;
DROP INDEX IF EXISTS public.idx_assistant_webhook_logs_webhook_id;

CREATE INDEX IF NOT EXISTS idx_assistant_http_logs_assistant_id
    ON public.assistant_http_logs USING btree (assistant_id);

CREATE INDEX IF NOT EXISTS idx_assistant_http_logs_organization_id
    ON public.assistant_http_logs USING btree (organization_id);

CREATE INDEX IF NOT EXISTS idx_assistant_http_logs_project_id
    ON public.assistant_http_logs USING btree (project_id);

CREATE INDEX IF NOT EXISTS idx_assistant_http_logs_assistant_conversation_id
    ON public.assistant_http_logs USING btree (assistant_conversation_id);

CREATE INDEX IF NOT EXISTS idx_assistant_http_logs_source_ref
    ON public.assistant_http_logs USING btree (source, source_ref_id);

CREATE INDEX IF NOT EXISTS idx_assistant_http_logs_created_date
    ON public.assistant_http_logs USING btree (created_date);
