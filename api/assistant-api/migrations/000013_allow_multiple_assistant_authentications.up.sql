ALTER TABLE public.assistant_authentications
    DROP CONSTRAINT IF EXISTS uk_assistant_authentications_assistant_id;

DROP INDEX IF EXISTS public.uk_assistant_authentications_assistant_id;
DROP INDEX IF EXISTS public.idx_assistant_authentications_assistant_id_unique;
