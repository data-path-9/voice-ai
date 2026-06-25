import { AssistantConfiguration } from '@rapidaai/react';
import { ColumnarType, PaginatedType } from '@/types';

export type AssistantTelemetryProperty = {
  telemetries: AssistantConfiguration[];
};

export type AssistantTelemetryType = {
  onChangeAssistantTelemetries: (telemetries: AssistantConfiguration[]) => void;
  getAssistantTelemetry: (
    assistantId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (telemetries: AssistantConfiguration[]) => void,
  ) => void;
  deleteAssistantTelemetry: (
    assistantId: string,
    telemetryId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (telemetry: AssistantConfiguration) => void,
  ) => void;
  updateAssistantTelemetryEnabled: (
    assistantId: string,
    telemetry: AssistantConfiguration,
    enabled: boolean,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (telemetry: AssistantConfiguration) => void,
  ) => void;
  clear: () => void;
} & AssistantTelemetryProperty &
  PaginatedType &
  ColumnarType;
