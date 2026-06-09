UPDATE public.assistant_provider_model_options options
SET key = 'listen.query_params'
FROM public.assistant_provider_models models
WHERE options.assistant_provider_model_id = models.id
  AND models.model_provider_name = 'custom-stt'
  AND options.key = 'listen.ws.query_params';

UPDATE public.assistant_provider_model_options options
SET key = 'listen.request_rules'
FROM public.assistant_provider_models models
WHERE options.assistant_provider_model_id = models.id
  AND models.model_provider_name = 'custom-stt'
  AND options.key = 'listen.ws.request_rules';

UPDATE public.assistant_provider_model_options options
SET key = 'listen.response_rules'
FROM public.assistant_provider_models models
WHERE options.assistant_provider_model_id = models.id
  AND models.model_provider_name = 'custom-stt'
  AND options.key = 'listen.ws.response_rules';
