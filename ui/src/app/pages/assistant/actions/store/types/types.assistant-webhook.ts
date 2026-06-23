import { AssistantConfiguration } from '@rapidaai/react';
import { ColumnarType, PaginatedType } from '@/types';

/**
 * assistant context
 */

export type AssistantWebhookProperty = {
  /**
   * list of activity log
   */
  webhooks: AssistantConfiguration[];
};

/**
 *
 */
export type AssistantWebhookType = {
  /**
   *
   * @param ep
   * @returns
   */
  onChangeAssistantWebhooks: (ep: AssistantConfiguration[]) => void;
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  getAssistantWebhook: (
    assistantId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: AssistantConfiguration[]) => void,
  ) => void;

  /**
   *
   * @param assistantId
   * @param webhookId
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  deleteAssistantWebhook: (
    assistantId: string,
    webhookId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: AssistantConfiguration) => void,
  ) => void;
  updateAssistantWebhookEnabled: (
    assistantId: string,
    webhook: AssistantConfiguration,
    enabled: boolean,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: AssistantConfiguration) => void,
  ) => void;
  /**
   * clear everything
   * @returns
   */
  clear: () => void;
} & AssistantWebhookProperty &
  PaginatedType &
  ColumnarType;
