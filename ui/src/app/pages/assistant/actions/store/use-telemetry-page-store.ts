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
  AssistantTelemetryProperty,
  AssistantTelemetryType,
} from './types/types.assistant-telemetry';
import { connectionConfig } from '@/configs';

const telemetryConfigurationType = 'telemetry';

const initialAssistantTelemetry: AssistantTelemetryProperty = {
  telemetries: [],
};

const errorText = (err: unknown, fallback: string) =>
  err instanceof Error ? err.message || fallback : fallback;

export const useAssistantTelemetryPageStore = create<AssistantTelemetryType>(
  (set, get) => ({
    ...initialAssistantTelemetry,
    ...initialPaginated,

    setPageSize: (pageSize: number) => {
      set({ page: 1, pageSize });
    },

    setPage: (pg: number) => {
      set({ page: pg });
    },

    setTotalCount: (tc: number) => {
      set({ totalCount: tc });
    },

    onChangeAssistantTelemetries: (telemetries: AssistantConfiguration[]) => {
      set({ telemetries });
    },

    addCriteria: (k: string, v: string, logic: string) => {
      let current = get().criteria.filter(
        x => x.key !== k && x.logic !== logic,
      );
      if (v) current.push({ key: k, value: v, logic: logic });
      set({ criteria: current });
    },

    addCriterias: (v: { k: string; v: string; logic: string }[]) => {
      let current = get().criteria.filter(
        x => !v.find(y => y.k === x.key && x.logic === y.logic),
      );
      v.forEach(c => {
        current.push({ key: c.k, value: c.v, logic: c.logic });
      });
      set({ criteria: current });
    },

    getAssistantTelemetry: (
      assistantId: string,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (telemetries: AssistantConfiguration[]) => void,
    ) => {
      const request = new GetAllAssistantConfigurationRequest();
      request.setAssistantid(assistantId);
      request.setConfigurationtype(telemetryConfigurationType);

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
            get().onChangeAssistantTelemetries(data);
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
              'Unable to get assistant telemetry, please try again later.',
          );
        })
        .catch(err => {
          onError(errorText(err, 'Unable to fetch assistant telemetry.'));
        });
    },

    deleteAssistantTelemetry: (
      assistantId: string,
      telemetryId: string,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (telemetry: AssistantConfiguration) => void,
    ) => {
      const request = new DeleteAssistantConfigurationRequest();
      request.setAssistantid(assistantId);
      request.setId(telemetryId);

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
              'Unable to delete assistant telemetry, please try again later.',
          );
        })
        .catch(err => {
          onError(errorText(err, 'Unable to delete assistant telemetry.'));
        });
    },

    updateAssistantTelemetryEnabled: (
      assistantId: string,
      telemetry: AssistantConfiguration,
      enabled: boolean,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (telemetry: AssistantConfiguration) => void,
    ) => {
      const request = new UpdateAssistantConfigurationRequest();
      request.setId(telemetry.getId());
      request.setAssistantid(assistantId);
      request.setConfigurationtype(telemetryConfigurationType);
      request.setProvider(telemetry.getProvider());
      request.setEnabled(enabled);
      request.setOptionsList(telemetry.getOptionsList());

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
              'Unable to update assistant telemetry, please try again later.',
          );
        })
        .catch(err => {
          onError(errorText(err, 'Unable to update assistant telemetry.'));
        });
    },

    columns: [
      { name: 'ID', key: 'id', visible: false },
      { name: 'Provider', key: 'provider', visible: true },
      { name: 'Target', key: 'target', visible: true },
      { name: 'Created Date', key: 'createdDate', visible: true },
    ],

    setColumns(cl: { name: string; key: string; visible: boolean }[]) {
      set({ columns: cl });
    },

    visibleColumn: (k: string): boolean => {
      const column = get().columns.find(c => c.key === k);
      return column ? column.visible : false;
    },

    clear: () => set({ ...initialAssistantTelemetry, ...initialPaginated }),
  }),
);
