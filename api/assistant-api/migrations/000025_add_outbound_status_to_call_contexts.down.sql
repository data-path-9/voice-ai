ALTER TABLE public.call_contexts
    DROP COLUMN IF EXISTS provider_status_code,
    DROP COLUMN IF EXISTS retryable,
    DROP COLUMN IF EXISTS disconnect_reason,
    DROP COLUMN IF EXISTS failure_reason,
    DROP COLUMN IF EXISTS failure_class,
    DROP COLUMN IF EXISTS call_error,
    DROP COLUMN IF EXISTS call_status;
