CREATE TABLE public.assistant_authentications (
    id bigint PRIMARY KEY,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    assistant_id bigint NOT NULL,
    fail_behavior character varying(20) NOT NULL DEFAULT 'block',
    timeout_ms bigint NOT NULL DEFAULT 5000
);

CREATE INDEX idx_assistant_authentications_assistant_id
    ON public.assistant_authentications (assistant_id);

CREATE INDEX idx_assistant_authentications_assistant_id_status
    ON public.assistant_authentications (assistant_id, status);

CREATE TABLE public.assistant_authentication_options (
    id bigint PRIMARY KEY,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    assistant_authentication_id bigint NOT NULL
);

ALTER TABLE ONLY public.assistant_authentication_options
    ADD CONSTRAINT uk_assistant_authentication_options UNIQUE (key, assistant_authentication_id);

CREATE INDEX idx_assistant_authentication_options_auth_id
    ON public.assistant_authentication_options (assistant_authentication_id);
