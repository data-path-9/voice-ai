DROP INDEX IF EXISTS public.idx_assistant_http_logs_created_date;
DROP INDEX IF EXISTS public.idx_assistant_http_logs_source_ref;
DROP INDEX IF EXISTS public.idx_assistant_http_logs_assistant_conversation_id;
DROP INDEX IF EXISTS public.idx_assistant_http_logs_project_id;
DROP INDEX IF EXISTS public.idx_assistant_http_logs_organization_id;
DROP INDEX IF EXISTS public.idx_assistant_http_logs_assistant_id;

-- assistant_webhook_logs stores only webhook logs. Drop non-webhook rows for rollback safety.
DELETE FROM public.assistant_http_logs
WHERE source <> 'webhook'
   OR source_ref_id = 0;

ALTER TABLE IF EXISTS public.assistant_http_logs
    DROP COLUMN IF EXISTS error_message,
    DROP COLUMN IF EXISTS context_id,
    DROP COLUMN IF EXISTS source;

ALTER TABLE IF EXISTS public.assistant_http_logs
    RENAME COLUMN source_ref_id TO webhook_id;

ALTER TABLE IF EXISTS public.assistant_http_logs
    RENAME COLUMN source_event TO event;

ALTER TABLE IF EXISTS public.assistant_http_logs
    ALTER COLUMN webhook_id SET NOT NULL;

ALTER TABLE IF EXISTS public.assistant_http_logs
    RENAME TO assistant_webhook_logs;

CREATE INDEX IF NOT EXISTS idx_assistant_webhook_logs_assistant_id
    ON public.assistant_webhook_logs USING btree (assistant_id);

CREATE INDEX IF NOT EXISTS idx_assistant_webhook_logs_organization_id
    ON public.assistant_webhook_logs USING btree (organization_id);

CREATE INDEX IF NOT EXISTS idx_assistant_webhook_logs_project_id
    ON public.assistant_webhook_logs USING btree (project_id);

CREATE INDEX IF NOT EXISTS idx_assistant_webhook_logs_webhook_id
    ON public.assistant_webhook_logs USING btree (webhook_id);
