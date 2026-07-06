import type { AgentflowNode } from '@/app/pages/assistant/actions/agentflow';

type StaticMessageNodeValidationContext = {
  getNodeValidationIssues: (node: AgentflowNode) => string[];
};

export const validateStaticMessageNode = (
  node: AgentflowNode,
  context: StaticMessageNodeValidationContext,
) => context.getNodeValidationIssues(node);
