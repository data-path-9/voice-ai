ALTER TABLE public.call_contexts
    ADD COLUMN IF NOT EXISTS call_status character varying(30) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS call_error text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS failure_class character varying(80) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS failure_reason text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS disconnect_reason character varying(120) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS retryable boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS provider_status_code integer NOT NULL DEFAULT 0;
