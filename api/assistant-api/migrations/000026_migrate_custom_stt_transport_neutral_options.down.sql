UPDATE public.assistant_deployment_audio_options options
SET key = 'listen.ws.query_params' WHERE options.key = 'listen.query_params';

UPDATE public.assistant_deployment_audio_options options
SET key = 'listen.ws.request_rules' WHERE options.key = 'listen.request_rules';

UPDATE public.assistant_deployment_audio_options options
SET key = 'listen.ws.response_rules' WHERE options.key = 'listen.response_rules';
