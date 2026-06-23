CREATE TABLE public.assistant_configurations (
    id bigint PRIMARY KEY,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    created_by bigint,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    project_id bigint NOT NULL,
    organization_id bigint NOT NULL,
    assistant_id bigint NOT NULL,
    configuration_type character varying(50) NOT NULL,
    provider character varying(50) NOT NULL
);

CREATE INDEX idx_assistant_configurations_assistant_id
    ON public.assistant_configurations USING btree (assistant_id);

CREATE INDEX idx_assistant_configurations_project_id
    ON public.assistant_configurations USING btree (project_id);

CREATE INDEX idx_assistant_configurations_organization_id
    ON public.assistant_configurations USING btree (organization_id);

CREATE INDEX idx_assistant_configurations_assistant_org_project
    ON public.assistant_configurations USING btree (assistant_id, organization_id, project_id);

CREATE INDEX idx_assistant_configurations_type_provider
    ON public.assistant_configurations USING btree (configuration_type, provider);

CREATE INDEX idx_assistant_configurations_runtime_lookup
    ON public.assistant_configurations USING btree (assistant_id, configuration_type, status, enabled);

CREATE TABLE public.assistant_configuration_options (
    id bigint PRIMARY KEY,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    assistant_configuration_id bigint NOT NULL REFERENCES public.assistant_configurations(id) ON DELETE CASCADE
);

ALTER TABLE ONLY public.assistant_configuration_options
    ADD CONSTRAINT uk_assistant_configuration_options UNIQUE (key, assistant_configuration_id);

CREATE INDEX idx_assistant_configuration_options_configuration_id
    ON public.assistant_configuration_options USING btree (assistant_configuration_id);
