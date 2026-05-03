ALTER TABLE public.assistant_authentications
    ADD CONSTRAINT uk_assistant_authentications_assistant_id UNIQUE (assistant_id);
