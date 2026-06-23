import { Metadata } from '@rapidaai/react';
import { ProviderComponentProps } from '@/app/components/providers';
import { loadProviderConfig } from '@/providers/config-loader';
import {
  getDefaultsFromConfig,
  validateFromConfig,
} from '@/providers/config-defaults';
import { ConfigRenderer } from '@/app/components/providers/config-renderer';
import { FC } from 'react';

export const GetDefaultTelemetryIfInvalid = (
  provider: string,
  parameters: Metadata[],
) => {
  const config = loadProviderConfig(provider);
  if (!config?.telemetry) return parameters;
  return getDefaultsFromConfig(config, 'telemetry', parameters, provider, {
    includeCredential: true,
  });
};

export const ValidateTelemetry = (
  provider: string,
  parameters: Metadata[],
): string | undefined => {
  const credentialId = parameters.find(
    p => p.getKey() === 'rapida.credential_id',
  );
  if (!credentialId || !credentialId.getValue()) {
    return `Please provide a valid ${provider} credential.`;
  }

  const config = loadProviderConfig(provider);
  if (!config?.telemetry) return undefined;
  return validateFromConfig(config, 'telemetry', provider, parameters);
};

export const TelemetryConfigComponent: FC<
  Pick<ProviderComponentProps, 'provider' | 'parameters' | 'onChangeParameter'>
> = ({ provider, parameters, onChangeParameter }) => {
  const config = loadProviderConfig(provider);
  if (!config?.telemetry) return null;
  return (
    <ConfigRenderer
      provider={provider}
      category="telemetry"
      config={config.telemetry}
      parameters={parameters}
      onParameterChange={onChangeParameter}
    />
  );
};
