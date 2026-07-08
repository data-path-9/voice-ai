import type { AgentflowNode } from '@/app/pages/assistant/actions/agentflow';

type ChatInputNodeValidationContext = {
  getNodeValidationIssues: (node: AgentflowNode) => string[];
};

export const validateChatInputNode = (
  node: AgentflowNode,
  context: ChatInputNodeValidationContext,
) => context.getNodeValidationIssues(node);
