UPDATE public.assistant_provider_model_options options
SET key = 'speak.query_params'
FROM public.assistant_provider_models models
WHERE options.assistant_provider_model_id = models.id
  AND models.model_provider_name = 'custom-tts'
  AND options.key = 'speak.ws.query_params';

UPDATE public.assistant_provider_model_options options
SET key = 'speak.request_rules'
FROM public.assistant_provider_models models
WHERE options.assistant_provider_model_id = models.id
  AND models.model_provider_name = 'custom-tts'
  AND options.key = 'speak.ws.request_rules';

UPDATE public.assistant_provider_model_options options
SET key = 'speak.response_rules'
FROM public.assistant_provider_models models
WHERE options.assistant_provider_model_id = models.id
  AND models.model_provider_name = 'custom-tts'
  AND options.key = 'speak.ws.response_rules';
