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
  return list.find(provider => provider.code === 'custom-stt');
}

describe('Custom STT provider catalog', () => {
  it('exists in development and production with websocket credential fields', () => {
    const developmentProviders = require('../provider.development.json');
    const productionProviders = require('../provider.production.json');

    const developmentProvider = getProvider(developmentProviders);
    const productionProvider = getProvider(productionProviders);

    for (const provider of [developmentProvider, productionProvider]) {
      expect(provider).toBeDefined();
      expect(provider.featureList).toEqual(
        expect.arrayContaining(['stt', 'external']),
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

describe('Custom STT config contract', () => {
  const config = loadProviderConfig('custom-stt')!;
  const validQueryParams =
    '{"language":{"$var":"language"},"sample_rate":{"$cast":"number","value":{"$var":"sample_rate"}}}';
  const validAudioRequest =
    '{"audio":{"$var":"audio"},"encoding":{"$var":"encoding"},"sample_rate":{"$cast":"number","value":{"$var":"sample_rate"}}}';
  const validResponseParser =
    '[{"when":{"frame":"text"},"emit":{"script":{"$frame":"text"},"language":"hi","interim":true}}]';
  const validJsonResponseParser =
    '[{"when":{"frame":"json","path":"type","equals":"partial"},"emit":{"script":{"$path":"text"},"interim":true}},{"when":{"frame":"json","path":"type","equals":"final"},"emit":{"script":{"$path":"text"},"confidence":{"$cast":"number","value":{"$path":"confidence"}},"language":{"$path":"language"},"interim":false}}]';

  const buildValidOptions = (): Metadata[] => [
    ...getDefaultsFromConfig(
      config,
      'stt',
      [createMetadata('rapida.credential_id', 'cred-custom-stt-1')],
      'custom-stt',
    ),
    createMetadata('listen.ws.query_params', validQueryParams),
    createMetadata('listen.ws.audio_request', validAudioRequest),
    createMetadata('listen.ws.response_parser', validResponseParser),
  ];

  it('loads the expected STT metadata keys', () => {
    expect(config.stt).toBeDefined();
    const keys = config.stt?.parameters.map(param => param.key) ?? [];
    expect(keys).toEqual(
      expect.arrayContaining([
        'listen.model',
        'listen.language',
        'listen.audio.encoding',
        'listen.audio.sample_rate',
        'listen.ws.query_params',
        'listen.ws.audio_request',
        'listen.ws.response_parser',
      ]),
    );
  });

  it('applies encoding and sample-rate defaults', () => {
    const defaults = getDefaultsFromConfig(
      config,
      'stt',
      [createMetadata('rapida.credential_id', 'cred-custom-stt-1')],
      'custom-stt',
    );

    expect(findMeta(defaults, 'listen.audio.encoding')).toBe('LINEAR16');
    expect(findMeta(defaults, 'listen.audio.sample_rate')).toBe('16000');
  });

  it.each([
    [
      'listen.audio.encoding',
      'Please select a valid audio encoding for custom STT.',
    ],
    [
      'listen.audio.sample_rate',
      'Please select a valid sample rate for custom STT.',
    ],
    [
      'listen.ws.response_parser',
      'Please provide a valid response parser for custom STT.',
    ],
  ])('requires %s', (key, expectedError) => {
    const result = validateFromConfig(
      config,
      'stt',
      'custom-stt',
      removeMetadata(buildValidOptions(), key),
    );

    expect(result).toBe(expectedError);
  });

  it('accepts a valid websocket STT DSL configuration', () => {
    expect(
      validateFromConfig(config, 'stt', 'custom-stt', buildValidOptions()),
    ).toBeUndefined();
  });

  it('also accepts JSON response parser rules', () => {
    const options = [
      ...buildValidOptions().filter(
        item => item.getKey() !== 'listen.ws.response_parser',
      ),
      createMetadata('listen.ws.response_parser', validJsonResponseParser),
    ];

    expect(validateFromConfig(config, 'stt', 'custom-stt', options)).toBeUndefined();
  });

  it('allows model, language, query params, and audio request to be omitted', () => {
    const options = buildValidOptions().filter(
      item =>
        ![
          'listen.model',
          'listen.language',
          'listen.ws.query_params',
          'listen.ws.audio_request',
        ].includes(item.getKey()),
    );

    expect(validateFromConfig(config, 'stt', 'custom-stt', options)).toBeUndefined();
  });

  it('rejects invalid JSON in optional query params', () => {
    const options = [
      ...buildValidOptions().filter(item => item.getKey() !== 'listen.ws.query_params'),
      createMetadata('listen.ws.query_params', '{"language":{"$var":"language"'),
    ];

    expect(validateFromConfig(config, 'stt', 'custom-stt', options)).toBe(
      'Please provide a valid JSON definition for query parameters.',
    );
  });

  it('rejects invalid JSON in the optional audio request', () => {
    const options = [
      ...buildValidOptions().filter(item => item.getKey() !== 'listen.ws.audio_request'),
      createMetadata('listen.ws.audio_request', '{"audio":{"$var":"audio"}'),
    ];

    expect(validateFromConfig(config, 'stt', 'custom-stt', options)).toBe(
      'Please provide a valid JSON definition for audio request.',
    );
  });

  it('rejects invalid response parser JSON', () => {
    const options = [
      ...buildValidOptions().filter(
        item => item.getKey() !== 'listen.ws.response_parser',
      ),
      createMetadata('listen.ws.response_parser', '[{"when":'),
    ];

    expect(validateFromConfig(config, 'stt', 'custom-stt', options)).toBe(
      'Please provide a valid JSON response parser for custom STT.',
    );
  });
});
