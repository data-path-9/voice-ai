import type { AgentflowNode } from '@/app/pages/assistant/actions/agentflow';

type ActionNodeValidationContext = {
  getNodeValidationIssues: (node: AgentflowNode) => string[];
};

export const validateActionNode = (
  node: AgentflowNode,
  context: ActionNodeValidationContext,
) => context.getNodeValidationIssues(node);
