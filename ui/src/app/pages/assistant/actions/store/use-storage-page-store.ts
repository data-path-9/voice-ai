import { create } from 'zustand';
import { initialPaginated } from '@/types/types.paginated';
import {
  AssistantConfiguration,
  DeleteAssistantConfiguration,
  DeleteAssistantConfigurationRequest,
  GetAllAssistantConfiguration,
  GetAllAssistantConfigurationRequest,
  GetAssistantConfigurationResponse,
  Paginate,
  UpdateAssistantConfiguration,
  UpdateAssistantConfigurationRequest,
} from '@rapidaai/react';
import {
  AssistantStorageProperty,
  AssistantStorageType,
} from './types/types.assistant-storage';
import { connectionConfig } from '@/configs';

const storageConfigurationType = 'storage';

const initialAssistantStorage: AssistantStorageProperty = {
  storages: [],
};

const errorText = (err: unknown, fallback: string) =>
  err instanceof Error ? err.message || fallback : fallback;

export const useAssistantStoragePageStore = create<AssistantStorageType>(
  (set, get) => ({
    ...initialAssistantStorage,
    ...initialPaginated,

    setPageSize: (pageSize: number) => {
      set({ page: 1, pageSize });
    },

    setPage: (page: number) => {
      set({ page });
    },

    setTotalCount: (totalCount: number) => {
      set({ totalCount });
    },

    onChangeAssistantStorages: (storages: AssistantConfiguration[]) => {
      set({ storages });
    },

    addCriteria: (key: string, value: string, logic: string) => {
      const current = get().criteria.filter(
        criteria => criteria.key !== key && criteria.logic !== logic,
      );
      if (value) current.push({ key, value, logic });
      set({ criteria: current });
    },

    addCriterias: (values: { k: string; v: string; logic: string }[]) => {
      const current = get().criteria.filter(
        criteria =>
          !values.find(v => v.k === criteria.key && criteria.logic === v.logic),
      );
      values.forEach(criteria => {
        current.push({
          key: criteria.k,
          value: criteria.v,
          logic: criteria.logic,
        });
      });
      set({ criteria: current });
    },

    getAssistantStorage: (
      assistantId: string,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (storages: AssistantConfiguration[]) => void,
    ) => {
      const request = new GetAllAssistantConfigurationRequest();
      request.setAssistantid(assistantId);
      request.setConfigurationtype(storageConfigurationType);

      const paginate = new Paginate();
      paginate.setPage(get().page);
      paginate.setPagesize(get().pageSize);
      request.setPaginate(paginate);

      GetAllAssistantConfiguration(connectionConfig, request, {
        authorization: token,
        'x-project-id': projectId,
        'x-auth-id': userId,
      })
        .then(response => {
          if (response?.getSuccess()) {
            const data = response.getDataList();
            get().onChangeAssistantStorages(data);
            const paginated = response.getPaginated();
            if (paginated) {
              get().setTotalCount(paginated.getTotalitem());
            }
            onSuccess(data);
            return;
          }

          const message = response?.getError()?.getHumanmessage();
          onError(
            message ||
              'Unable to get assistant storage, please try again later.',
          );
        })
        .catch(err => {
          onError(errorText(err, 'Unable to fetch assistant storage.'));
        });
    },

    deleteAssistantStorage: (
      assistantId: string,
      storageId: string,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (storage: AssistantConfiguration) => void,
    ) => {
      const request = new DeleteAssistantConfigurationRequest();
      request.setAssistantid(assistantId);
      request.setId(storageId);

      DeleteAssistantConfiguration(connectionConfig, request, {
        authorization: token,
        'x-project-id': projectId,
        'x-auth-id': userId,
      })
        .then((response: GetAssistantConfigurationResponse) => {
          if (response?.getSuccess() && response.getData()) {
            onSuccess(response.getData()!);
            return;
          }

          const message = response?.getError()?.getHumanmessage();
          onError(
            message ||
              'Unable to delete assistant storage, please try again later.',
          );
        })
        .catch(err => {
          onError(errorText(err, 'Unable to delete assistant storage.'));
        });
    },

    updateAssistantStorageEnabled: (
      assistantId: string,
      storage: AssistantConfiguration,
      enabled: boolean,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (storage: AssistantConfiguration) => void,
    ) => {
      const request = new UpdateAssistantConfigurationRequest();
      request.setId(storage.getId());
      request.setAssistantid(assistantId);
      request.setConfigurationtype(storageConfigurationType);
      request.setProvider(storage.getProvider());
      request.setEnabled(enabled);
      request.setOptionsList(storage.getOptionsList());

      UpdateAssistantConfiguration(connectionConfig, request, {
        authorization: token,
        'x-project-id': projectId,
        'x-auth-id': userId,
      })
        .then((response: GetAssistantConfigurationResponse) => {
          if (response?.getSuccess() && response.getData()) {
            onSuccess(response.getData()!);
            return;
          }

          const message = response?.getError()?.getHumanmessage();
          onError(
            message ||
              'Unable to update assistant storage, please try again later.',
          );
        })
        .catch(err => {
          onError(errorText(err, 'Unable to update assistant storage.'));
        });
    },

    columns: [
      { name: 'ID', key: 'id', visible: false },
      { name: 'Provider', key: 'provider', visible: true },
      { name: 'Target', key: 'target', visible: true },
      { name: 'Created Date', key: 'createdDate', visible: true },
    ],

    setColumns(columns: { name: string; key: string; visible: boolean }[]) {
      set({ columns });
    },

    visibleColumn: (key: string): boolean => {
      const column = get().columns.find(c => c.key === key);
      return column ? column.visible : false;
    },

    clear: () => set({ ...initialAssistantStorage, ...initialPaginated }),
  }),
);
