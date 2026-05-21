export interface AIBlock {
  type: string;       // paragraph, heading1, heading2, heading3, bulleted_list_item, etc.
  content: string;
}

export interface WritingRequest {
  action: string;     // generate, rewrite, summarize, expand, translate, proofread
  context: string;
  prompt: string;
  tone?: string;
  lang?: string;
}

export interface WritingResponse {
  blocks: AIBlock[];
  usage: { prompt_tokens: number; completion_tokens: number; total_tokens: number };
}

export interface QARequest {
  question: string;
  workspace_id: number;
}

export interface Citation {
  page_id: number;
  block_id?: number;
  title: string;
  snippet: string;
}

export interface QAResponse {
  answer: string;
  citations: Citation[];
  usage: { prompt_tokens: number; completion_tokens: number; total_tokens: number };
}

export interface AutofillRequest {
  database_id: number;
  property_id: number;
  source_prop_id: number;
  record_ids?: number[];
  instruction?: string;
}

export interface AutofillJob {
  id: string;
  database_id: number;
  property_id: number;
  total: number;
  completed: number;
  failed: number;
  status: 'pending' | 'running' | 'completed' | 'failed';
}

export interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
  citations?: Citation[];
}
