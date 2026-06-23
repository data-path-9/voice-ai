import { AssistantConfiguration } from '@rapidaai/react';
import { ColumnarType, PaginatedType } from '@/types';

export type AssistantStorageProperty = {
  storages: AssistantConfiguration[];
};

export type AssistantStorageType = {
  onChangeAssistantStorages: (storages: AssistantConfiguration[]) => void;
  getAssistantStorage: (
    assistantId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (storages: AssistantConfiguration[]) => void,
  ) => void;
  deleteAssistantStorage: (
    assistantId: string,
    storageId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (storage: AssistantConfiguration) => void,
  ) => void;
  updateAssistantStorageEnabled: (
    assistantId: string,
    storage: AssistantConfiguration,
    enabled: boolean,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (storage: AssistantConfiguration) => void,
  ) => void;
  clear: () => void;
} & AssistantStorageProperty &
  PaginatedType &
  ColumnarType;
