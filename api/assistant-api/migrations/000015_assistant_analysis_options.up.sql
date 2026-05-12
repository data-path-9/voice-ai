CREATE TABLE public.assistant_analysis_options (
    id bigint PRIMARY KEY,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    assistant_analysis_id bigint NOT NULL
);

ALTER TABLE ONLY public.assistant_analysis_options
    ADD CONSTRAINT uk_assistant_analysis_option UNIQUE (key, assistant_analysis_id);

CREATE INDEX idx_assistant_analysis_options_assistant_analysis_id
    ON public.assistant_analysis_options USING btree (assistant_analysis_id);
