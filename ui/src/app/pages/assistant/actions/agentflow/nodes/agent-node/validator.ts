import type { AgentflowNode } from '@/app/pages/assistant/actions/agentflow';

type AgentNodeValidationContext = {
  getNodeValidationIssues: (node: AgentflowNode) => string[];
};

export const validateAgentNode = (
  node: AgentflowNode,
  context: AgentNodeValidationContext,
) => context.getNodeValidationIssues(node);
