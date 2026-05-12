import { AssistantHTTPLog } from '@rapidaai/react';
import { ColumnarType } from './types.columnar';
import { PaginatedType } from './types.paginated';

/**
 * assistant context
 */

export type WebhookLogTypeProperty = {
  /**
   * list of Webhook log
   */
  webhookLogs: AssistantHTTPLog[];
};

/**
 *
 */
export type WebhookLogType = {
  /**
   *
   * @param ep
   * @returns
   */
  onChangeActivities: (ep: AssistantHTTPLog[]) => void;
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  getActivities: (
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: AssistantHTTPLog[]) => void,
  ) => void;

  /**
   * clear everything
   * @returns
   */
  clear: () => void;
} & WebhookLogTypeProperty &
  PaginatedType &
  ColumnarType;
