import { AssistantConfiguration } from '@rapidaai/react';
import { ColumnarType, PaginatedType } from '@/types';

/**
 * assistant context
 */

export type AssistantAnalysisProperty = {
  /**
   * list of activity log
   */
  analysises: AssistantConfiguration[];
};

/**
 *
 */
export type AssistantAnalysisType = {
  /**
   *
   * @param ep
   * @returns
   */
  onChangeAssistantAnalysises: (ep: AssistantConfiguration[]) => void;
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  getAssistantAnalysis: (
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
   * @param analysisId
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  deleteAssistantAnalysis: (
    assistantId: string,
    analysisId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: AssistantConfiguration) => void,
  ) => void;
  updateAssistantAnalysisEnabled: (
    assistantId: string,
    analysis: AssistantConfiguration,
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
} & AssistantAnalysisProperty &
  PaginatedType &
  ColumnarType;
