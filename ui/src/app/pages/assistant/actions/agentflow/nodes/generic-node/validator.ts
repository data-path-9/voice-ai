import type { AgentflowNode } from '@/app/pages/assistant/actions/agentflow';

type GenericNodeValidationContext = {
  getNodeValidationIssues: (node: AgentflowNode) => string[];
};

export const validateGenericNode = (
  node: AgentflowNode,
  context: GenericNodeValidationContext,
) => context.getNodeValidationIssues(node);
