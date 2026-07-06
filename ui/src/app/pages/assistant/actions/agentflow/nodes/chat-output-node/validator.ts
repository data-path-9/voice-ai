import type { AgentflowNode } from '@/app/pages/assistant/actions/agentflow';

type ChatOutputNodeValidationContext = {
  getNodeValidationIssues: (node: AgentflowNode) => string[];
};

export const validateChatOutputNode = (
  node: AgentflowNode,
  context: ChatOutputNodeValidationContext,
) => context.getNodeValidationIssues(node);
