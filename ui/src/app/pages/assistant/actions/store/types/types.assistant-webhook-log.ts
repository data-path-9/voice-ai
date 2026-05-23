import { AssistantHTTPLog } from '@rapidaai/react';
import { ColumnarType } from '@/types/types.columnar';
import { PaginatedType } from '@/types/types.paginated';

/**
 * assistant context
 */

export type AssistantWebhookLogProperty = {
  /**
   * list of activity log
   */
  webhookLogs: AssistantHTTPLog[];
};

/**
 *
 */
export type AssistantWebhookLogType = {
  /**
   *
   * @param ep
   * @returns
   */
  onChangeAssistantWebhookLogs: (ep: AssistantHTTPLog[]) => void;
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  getAssistantWebhookLogs: (
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
} & AssistantWebhookLogProperty &
  PaginatedType &
  ColumnarType;
