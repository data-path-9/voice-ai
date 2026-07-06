import type { AgentflowNode } from '@/app/pages/assistant/actions/agentflow';

type ConditionNodeValidationContext = {
  getNodeValidationIssues: (node: AgentflowNode) => string[];
};

export const validateConditionNode = (
  node: AgentflowNode,
  context: ConditionNodeValidationContext,
) => context.getNodeValidationIssues(node);
