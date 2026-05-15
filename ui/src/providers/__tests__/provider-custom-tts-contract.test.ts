import { Metadata } from '@rapidaai/react';
import { loadProviderConfig } from '../config-loader';
import { getDefaultsFromConfig, validateFromConfig } from '../config-defaults';

function createMetadata(key: string, value: string): Metadata {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
}

function findMeta(source: Metadata[], key: string): string | undefined {
  return source.find(item => item.getKey() === key)?.getValue();
}

function removeMetadata(source: Metadata[], key: string): Metadata[] {
  return source.filter(item => item.getKey() !== key);
}

function getProvider(list: any[]) {
  return list.find(provider => provider.code === 'custom-tts');
}

describe('Custom TTS provider catalog', () => {
  it('exists in development and production with websocket credential fields', () => {
    const developmentProviders = require('../provider.development.json');
    const productionProviders = require('../provider.production.json');

    const developmentProvider = getProvider(developmentProviders);
    const productionProvider = getProvider(productionProviders);

    for (const provider of [developmentProvider, productionProvider]) {
      expect(provider).toBeDefined();
      expect(provider.featureList).toEqual(
        expect.arrayContaining(['tts', 'external']),
      );
      expect(provider.configurations.map((config: any) => config.name)).toEqual(
        expect.arrayContaining(['apiCompatibility', 'baseUrl', 'headers']),
      );
      const compatibilityConfig = provider.configurations.find(
        (config: any) => config.name === 'apiCompatibility',
      );
      expect(
        compatibilityConfig?.choices?.map((choice: any) => choice.value),
      ).toEqual(['websocket_v1']);
    }
  });
});

describe('Custom TTS TTS config contract', () => {
  const config = loadProviderConfig('custom-tts')!;
  const validQueryParams =
    '{"language":{"$var":"language"},"message_id":{"$var":"message_id"},"sample_rate":{"$cast":"number","value":{"$var":"sample_rate"}}}';
  const validTextRequest =
    '{"text":{"$var":"text"},"voice_id":{"$var":"voice_id"},"message_id":{"$var":"message_id"},"model":{"$var":"model"},"language":{"$var":"language"},"audio":{"encoding":{"$var":"encoding"},"sample_rate":{"$cast":"number","value":{"$var":"sample_rate"}}}}';
  const validResponseParser =
    '[{"when":{"frame":"binary"},"emit":{"audio":{"$frame":"binary"}}},{"when":{"frame":"json","path":"type","equals":"done"},"emit":{"message_id":{"$path":"message_id"},"done":true}}]';

  const buildValidOptions = (): Metadata[] => [
    ...getDefaultsFromConfig(
      config,
      'tts',
      [createMetadata('rapida.credential_id', 'cred-custom-1')],
      'custom-tts',
    ),
    createMetadata('speak.voice.id', 'narrator-1'),
    createMetadata('speak.ws.query_params', validQueryParams),
    createMetadata('speak.ws.text_request', validTextRequest),
    createMetadata('speak.ws.response_parser', validResponseParser),
  ];

  it('loads the expected TTS metadata keys', () => {
    expect(config.tts).toBeDefined();
    const keys = config.tts?.parameters.map(param => param.key) ?? [];
    expect(keys).toEqual(
      expect.arrayContaining([
        'speak.model',
        'speak.language',
        'speak.voice.id',
        'speak.audio.encoding',
        'speak.audio.sample_rate',
        'speak.ws.query_params',
        'speak.ws.text_request',
        'speak.ws.done_request',
        'speak.ws.response_parser',
      ]),
    );
  });

  it('applies encoding and sample-rate defaults', () => {
    const defaults = getDefaultsFromConfig(
      config,
      'tts',
      [createMetadata('rapida.credential_id', 'cred-custom-1')],
      'custom-tts',
    );

    expect(findMeta(defaults, 'speak.audio.encoding')).toBe('LINEAR16');
    expect(findMeta(defaults, 'speak.audio.sample_rate')).toBe('16000');
  });

  it.each([
    ['speak.voice.id', 'Please provide a valid voice ID for custom TTS.'],
    [
      'speak.audio.encoding',
      'Please select a valid audio encoding for custom TTS.',
    ],
    [
      'speak.audio.sample_rate',
      'Please select a valid sample rate for custom TTS.',
    ],
    [
      'speak.ws.text_request',
      'Please provide a valid text request for custom TTS.',
    ],
    [
      'speak.ws.response_parser',
      'Please provide a valid response parser for custom TTS.',
    ],
  ])('requires %s', (key, expectedError) => {
    const result = validateFromConfig(
      config,
      'tts',
      'custom-tts',
      removeMetadata(buildValidOptions(), key),
    );

    expect(result).toBe(expectedError);
  });

  it('accepts a valid one-shot websocket DSL configuration', () => {
    expect(
      validateFromConfig(config, 'tts', 'custom-tts', buildValidOptions()),
    ).toBeUndefined();
  });

  it('allows model and language to be omitted', () => {
    const options = buildValidOptions().filter(
      item =>
        item.getKey() !== 'speak.model' && item.getKey() !== 'speak.language',
    );

    expect(validateFromConfig(config, 'tts', 'custom-tts', options)).toBeUndefined();
  });

  it('rejects invalid JSON in optional query params', () => {
    const options = [
      ...buildValidOptions().filter(item => item.getKey() !== 'speak.ws.query_params'),
      createMetadata('speak.ws.query_params', '{"language":{"$var":"language"'),
    ];

    expect(validateFromConfig(config, 'tts', 'custom-tts', options)).toBe(
      'Please provide a valid JSON definition for query parameters.',
    );
  });

  it('rejects invalid JSON in the text request', () => {
    const options = buildValidOptions();
    const invalidTextRequest = createMetadata(
      'speak.ws.text_request',
      '{"text":{"$var":"text"}',
    );
    const replaced = [
      ...removeMetadata(options, 'speak.ws.text_request'),
      invalidTextRequest,
    ];

    expect(validateFromConfig(config, 'tts', 'custom-tts', replaced)).toBe(
      'Please provide a valid JSON definition for text request.',
    );
  });

  it('rejects invalid JSON in the optional done request', () => {
    const options = [
      ...buildValidOptions(),
      createMetadata('speak.ws.done_request', '{"type":"done",'),
    ];

    expect(validateFromConfig(config, 'tts', 'custom-tts', options)).toBe(
      'Please provide a valid JSON definition for done request.',
    );
  });

  it('rejects invalid response parser JSON', () => {
    const options = [
      ...buildValidOptions().filter(
        item => item.getKey() !== 'speak.ws.response_parser',
      ),
      createMetadata('speak.ws.response_parser', '[{"when":'),
    ];

    expect(validateFromConfig(config, 'tts', 'custom-tts', options)).toBe(
      'Please provide a valid JSON response parser for custom TTS.',
    );
  });
});
