ALTER TABLE public.assistant_authentications
    DROP COLUMN IF EXISTS provider;

ALTER TABLE public.assistant_webhooks
    DROP COLUMN IF EXISTS provider;

ALTER TABLE public.assistant_analyses
    DROP COLUMN IF EXISTS provider;
