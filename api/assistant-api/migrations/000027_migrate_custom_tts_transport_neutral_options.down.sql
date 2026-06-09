UPDATE public.assistant_deployment_audio_options options
SET key = 'speak.ws.query_params' WHERE options.key = 'speak.query_params';

UPDATE public.assistant_deployment_audio_options options
SET key = 'speak.ws.request_rules' WHERE options.key = 'speak.request_rules';

UPDATE public.assistant_deployment_audio_options options
SET key = 'speak.ws.response_rules' WHERE options.key = 'speak.response_rules';
