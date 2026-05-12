ALTER TABLE public.assistant_authentications
    ADD COLUMN IF NOT EXISTS provider character varying(50);

UPDATE public.assistant_authentications
SET provider = 'http'
WHERE provider IS NULL OR btrim(provider) = '' OR lower(btrim(provider)) <> 'http';

ALTER TABLE public.assistant_authentications
    ALTER COLUMN provider SET DEFAULT 'http',
    ALTER COLUMN provider SET NOT NULL;


ALTER TABLE public.assistant_webhooks
    ADD COLUMN IF NOT EXISTS provider character varying(50);

UPDATE public.assistant_webhooks
SET provider = 'http'
WHERE provider IS NULL OR btrim(provider) = '' OR lower(btrim(provider)) <> 'http';

ALTER TABLE public.assistant_webhooks
    ALTER COLUMN provider SET DEFAULT 'http',
    ALTER COLUMN provider SET NOT NULL;


ALTER TABLE public.assistant_analyses
    ADD COLUMN IF NOT EXISTS provider character varying(50);

UPDATE public.assistant_analyses
SET provider = 'endpoint'
WHERE provider IS NULL OR btrim(provider) = '' OR lower(btrim(provider)) <> 'endpoint';

ALTER TABLE public.assistant_analyses
    ALTER COLUMN provider SET DEFAULT 'endpoint',
    ALTER COLUMN provider SET NOT NULL;
