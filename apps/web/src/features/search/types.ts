export interface SearchResults {
  pages: SearchPage[];
  blocks: SearchBlock[];
  records: SearchRecord[];
}

export interface SearchPage {
  id: number;
  title: string;
  workspace_id: number;
}

export interface SearchBlock {
  id: number;
  text: string;
  page_id: number;
  workspace_id: number;
  block_type: string;
}

export interface SearchRecord {
  id: number;
  title: string;
  database_id: number;
  workspace_id: number;
}
