UPDATE public.assistant_deployment_audio_options options
SET key = 'listen.query_params' WHERE options.key = 'listen.ws.query_params';

UPDATE public.assistant_deployment_audio_options options
SET key = 'listen.request_rules' WHERE options.key = 'listen.ws.request_rules';

UPDATE public.assistant_deployment_audio_options options
SET key = 'listen.response_rules' WHERE options.key = 'listen.ws.response_rules';
