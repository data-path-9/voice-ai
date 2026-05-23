import { create } from 'zustand';
import {} from '@/types/types.activity-log';
import { initialPaginated } from '@/types/types.paginated';
import {
  WebhookLogType,
  WebhookLogTypeProperty,
} from '@/types/types.webhook-log';
import {
  AssistantHTTPLog,
  Criteria,
  GetAllAssistantHTTPLogRequest,
  Paginate,
} from '@rapidaai/react';
import { GetAllHTTPLog } from '@rapidaai/react';
import { connectionConfig } from '@/configs';

const intialActivityLog: WebhookLogTypeProperty = {
  webhookLogs: [],
};

/**
 *
 */
export const useWebhookLogPage = create<WebhookLogType>((set, get) => ({
  ...intialActivityLog,
  ...initialPaginated,

  /**
   *
   * @param number
   * @returns
   */
  setPageSize: (pageSize: number) => {
    // when someone change pagesize change the page to zero
    set({
      page: 1,
      pageSize: pageSize,
    });
  },

  /**
   *
   * @param number
   * @returns
   */
  setPage: (pg: number) => {
    set({
      page: pg,
    });
  },

  /**
   *
   * @param number
   * @returns
   */
  setTotalCount: (tc: number) => {
    set({
      totalCount: tc,
    });
  },

  /**
   *
   * @param ep
   */
  onChangeActivities: (lgs: AssistantHTTPLog[]) => {
    set({
      webhookLogs: lgs,
    });
  },

  /**
   *
   * @param k
   * @param v
   */
  addCriteria: (k: string, v: string, logic: string) => {
    let current = get().criteria.filter(x => x.key !== k && x.logic !== logic);
    if (v) current.push({ key: k, value: v, logic: logic });
    set({
      criteria: current,
    });
  },

  /**
   *
   * @param v
   */
  addCriterias: (v: { k: string; v: string; logic: string }[]) => {
    let current = get().criteria.filter(
      x => !v.find(y => y.k === x.key && x.logic === y.logic),
    );
    v.forEach(c => {
      current.push({ key: c.k, value: c.v, logic: c.logic });
    });
    set({
      criteria: current,
    });
  },

  /**
   *
   * @param projectId
   * @param token
   * @param userId
   */
  getActivities: async (
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: AssistantHTTPLog[]) => void,
  ) => {
    const req = new GetAllAssistantHTTPLogRequest();
    req.setProjectid(projectId);

    const paginate = new Paginate();
    paginate.setPage(get().page);
    paginate.setPagesize(get().pageSize);
    req.setPaginate(paginate);

    get().criteria.forEach(({ key, value, logic }) => {
      const ctr = new Criteria();
      ctr.setKey(key);
      ctr.setValue(value);
      ctr.setLogic(logic);
      req.addCriterias(ctr);
    });

    try {
      const gur = await GetAllHTTPLog(connectionConfig, req, {
        authorization: token,
        'x-project-id': projectId,
        'x-auth-id': userId,
      });

      if (gur?.getSuccess()) {
        get().onChangeActivities(gur.getDataList());
        let paginated = gur.getPaginated();
        if (paginated) {
          get().setTotalCount(paginated.getTotalitem());
        }
        onSuccess(gur.getDataList());
      } else {
        let errorMessage = gur?.getError();
        if (errorMessage) {
          onError(errorMessage.getHumanmessage());
          return;
        }
        onError('Unable to get request logs, please try again later.');
      }
    } catch {
      onError('Unable to get request logs, please try again later.');
    }
  },

  /**
   *   id: string,
       webhookid: string,
       request?: google_protobuf_struct_pb.Struct.AsObject,
       response?: google_protobuf_struct_pb.Struct.AsObject,
       status: string,
       createddate?: google_protobuf_timestamp_pb.Timestamp.AsObject,
       updateddate?: google_protobuf_timestamp_pb.Timestamp.AsObject,
       assistantid: string,
       projectid: string,
       organizationid: string,
       conversationid: string,
       assetprefix: string,
       event: string,
       responsestatus: string,
       timetaken: string,
       retrycount: number,
   * columns
   */
  columns: [
    { name: 'Request ID', key: 'sourcerefid', visible: true },
    { name: 'Session ID', key: 'sessionid', visible: true },
    { name: 'Event', key: 'event', visible: true },
    { name: 'Endpoint', key: 'endpoint', visible: true },
    { name: 'Actions', key: 'action', visible: true },
    { name: 'Http status', key: 'responsestatus', visible: true },
    { name: 'Time Taken', key: 'timetaken', visible: true },
    { name: 'Retry Count', key: 'retrycount', visible: true },
    { name: 'Date', key: 'created_date', visible: true },
  ],

  /**
   *
   * @param cl
   */
  setColumns(cl: { name: string; key: string; visible: boolean }[]) {
    set({
      columns: cl,
    });
  },

  /**
   *
   * @param k
   * @returns
   */
  visibleColumn: (k: string): boolean => {
    const column = get().columns.find(c => c.key === k);
    return column ? column.visible : false;
  },

  /**
   * clear everything from the context
   * @returns
   */
  clear: () => set({ ...intialActivityLog }, true),
}));
