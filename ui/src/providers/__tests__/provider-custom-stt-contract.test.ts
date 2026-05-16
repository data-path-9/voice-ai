import { Metadata } from '@rapidaai/react';
import { loadProviderConfig } from '../config-loader';
import { getDefaultsFromConfig, validateFromConfig } from '../config-defaults';
import {
  CUSTOM_STT_DEFAULT_REQUEST_RULES_EXAMPLE,
  CUSTOM_STT_REQUEST_RULES_KEY,
  CUSTOM_STT_RESPONSE_RULES_KEY,
} from '../custom-stt/contract';

function createMetadata(key: string, value: string): Metadata {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
}

function findMeta(source: Metadata[], key: string): string | undefined {
  return source.find(item => item.getKey() === key)?.getValue();
}

function upsertMetadata(
  source: Metadata[],
  key: string,
  value: string,
): Metadata[] {
  const next = source.filter(item => item.getKey() !== key);
  next.push(createMetadata(key, value));
  return next;
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
  const validJsonRequestRules =
    '[{"when":{"packet":"audio"},"send":{"frame":"json","body":{"audio":{"$path":"packet.audio.base64"},"encoding":{"$path":"config.audio.encoding"},"sample_rate":{"$cast":"number","value":{"$path":"config.audio.sample_rate"}}}}}]';
  const validResponseRules =
    '[{"when":{"frame":"text"},"emit":{"script":{"$frame":"text"},"language":"hi","interim":true}}]';
  const validJsonResponseRules =
    '[{"when":{"frame":"json","path":"type","equals":"partial"},"emit":{"script":{"$path":"text"},"interim":true}},{"when":{"frame":"json","path":"type","equals":"final"},"emit":{"script":{"$path":"text"},"confidence":{"$cast":"number","value":{"$path":"confidence"}},"language":{"$path":"language"},"interim":false}}]';

  const buildValidOptions = (): Metadata[] => {
    const defaults = getDefaultsFromConfig(
      config,
      'stt',
      [createMetadata('rapida.credential_id', 'cred-custom-stt-1')],
      'custom-stt',
    );

    return upsertMetadata(
      upsertMetadata(defaults, 'listen.ws.query_params', validQueryParams),
      CUSTOM_STT_RESPONSE_RULES_KEY,
      validResponseRules,
    );
  };

  it('loads the expected canonical STT metadata keys', () => {
    expect(config.stt).toBeDefined();
    const keys = config.stt?.parameters.map(param => param.key) ?? [];
    expect(keys).toEqual(
      expect.arrayContaining([
        'listen.model',
        'listen.language',
        'listen.audio.encoding',
        'listen.audio.sample_rate',
        'listen.ws.query_params',
        CUSTOM_STT_REQUEST_RULES_KEY,
        CUSTOM_STT_RESPONSE_RULES_KEY,
      ]),
    );
    expect(keys).not.toContain('listen.ws.audio_request');
    expect(keys).not.toContain('listen.ws.response_parser');
  });

  it('applies encoding, sample-rate, and request-rule defaults', () => {
    const defaults = getDefaultsFromConfig(
      config,
      'stt',
      [createMetadata('rapida.credential_id', 'cred-custom-stt-1')],
      'custom-stt',
    );

    expect(findMeta(defaults, 'listen.audio.encoding')).toBe('LINEAR16');
    expect(findMeta(defaults, 'listen.audio.sample_rate')).toBe('16000');
    expect(findMeta(defaults, CUSTOM_STT_REQUEST_RULES_KEY)).toBe(
      CUSTOM_STT_DEFAULT_REQUEST_RULES_EXAMPLE,
    );
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
      CUSTOM_STT_REQUEST_RULES_KEY,
      'Please provide valid request rules for custom STT.',
    ],
    [
      CUSTOM_STT_RESPONSE_RULES_KEY,
      'Please provide valid response rules for custom STT.',
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

  it('accepts a valid canonical websocket STT DSL configuration', () => {
    expect(
      validateFromConfig(config, 'stt', 'custom-stt', buildValidOptions()),
    ).toBeUndefined();
  });

  it('also accepts JSON request and response rules', () => {
    const options = upsertMetadata(
      upsertMetadata(
        buildValidOptions(),
        CUSTOM_STT_REQUEST_RULES_KEY,
        validJsonRequestRules,
      ),
      CUSTOM_STT_RESPONSE_RULES_KEY,
      validJsonResponseRules,
    );

    expect(
      validateFromConfig(config, 'stt', 'custom-stt', options),
    ).toBeUndefined();
  });

  it('allows model, language, and query params to be omitted', () => {
    const options = buildValidOptions().filter(
      item =>
        !['listen.model', 'listen.language', 'listen.ws.query_params'].includes(
          item.getKey(),
        ),
    );

    expect(
      validateFromConfig(config, 'stt', 'custom-stt', options),
    ).toBeUndefined();
  });

  it('rejects invalid JSON in optional query params', () => {
    const options = upsertMetadata(
      removeMetadata(buildValidOptions(), 'listen.ws.query_params'),
      'listen.ws.query_params',
      '{"language":{"$var":"language"',
    );

    expect(validateFromConfig(config, 'stt', 'custom-stt', options)).toBe(
      'Please provide a valid JSON definition for query parameters.',
    );
  });

  it('rejects invalid request rules JSON', () => {
    const options = upsertMetadata(
      removeMetadata(buildValidOptions(), CUSTOM_STT_REQUEST_RULES_KEY),
      CUSTOM_STT_REQUEST_RULES_KEY,
      '[{"when":{"packet":"audio"}}',
    );

    expect(validateFromConfig(config, 'stt', 'custom-stt', options)).toBe(
      'Please provide valid JSON request rules for custom STT.',
    );
  });

  it('rejects request rules without an audio packet rule', () => {
    const options = upsertMetadata(
      removeMetadata(buildValidOptions(), CUSTOM_STT_REQUEST_RULES_KEY),
      CUSTOM_STT_REQUEST_RULES_KEY,
      '[{"when":{"packet":"turn_change"},"send":{"frame":"json","body":{"type":"start"}}}]',
    );

    expect(validateFromConfig(config, 'stt', 'custom-stt', options)).toBe(
      'Custom STT request rules must contain at least one rule with when.packet "audio".',
    );
  });

  it('rejects invalid response rules JSON', () => {
    const options = upsertMetadata(
      removeMetadata(buildValidOptions(), CUSTOM_STT_RESPONSE_RULES_KEY),
      CUSTOM_STT_RESPONSE_RULES_KEY,
      '[{"when":',
    );

    expect(validateFromConfig(config, 'stt', 'custom-stt', options)).toBe(
      'Please provide a valid JSON response rules for custom STT.',
    );
  });
});
