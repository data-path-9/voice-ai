UPDATE public.assistant_deployment_audio_options options
SET key = 'speak.query_params' WHERE options.key = 'speak.ws.query_params';

UPDATE public.assistant_deployment_audio_options options
SET key = 'speak.request_rules' WHERE options.key = 'speak.ws.request_rules';

UPDATE public.assistant_deployment_audio_options options
SET key = 'speak.response_rules' WHERE options.key = 'speak.ws.response_rules';
