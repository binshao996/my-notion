// Core types matching Go structs

export interface Database {
  id: number;
  workspace_id: number;
  page_id: number;
  name: string;
  created_at: string;
  updated_at: string;
}

export type PropertyType =
  | "title" | "text" | "number" | "select"
  | "multi_select" | "status" | "date" | "person"
  | "files" | "checkbox" | "url" | "email" | "phone";

export interface SelectOption {
  id: string;
  name: string;
  color: string;
}

export interface PropertyConfig {
  options?: SelectOption[];
  format?: string;
}

export interface Property {
  id: number;
  database_id: number;
  name: string;
  type: PropertyType;
  config: PropertyConfig;
  position: string;
  created_at: string;
}

export interface RecordValue {
  record_id: number;
  property_id: number;
  value: any;
}

export interface Record {
  id: number;
  database_id: number;
  page_id: number;
  position: string;
  property_values?: RecordValue[];
  created_at: string;
}

export type ViewType = "table" | "board" | "calendar" | "list" | "gallery";

export type FilterOperator =
  | "equals" | "not_equals" | "contains" | "not_contains"
  | "is_empty" | "is_not_empty" | "greater_than" | "less_than"
  | "greater_than_or_equal" | "less_than_or_equal"
  | "starts_with" | "ends_with";

export interface FilterCondition {
  property_id: number;
  operator: FilterOperator;
  value: any;
}

export interface FilterGroup {
  and?: FilterCondition[];
  or?: FilterCondition[];
}

export interface SortConfig {
  property_id: number;
  direction: "asc" | "desc";
}

export interface GroupConfig {
  property_id: number;
}

export interface LayoutConfig {
  card_preview?: "none" | "page_cover" | "page_content";
  card_size?: "small" | "medium" | "large";
  fit_to_page?: boolean;
}

export interface ViewConfig {
  filters?: FilterGroup;
  sorts?: SortConfig[];
  groupBy?: GroupConfig;
  hidden_properties?: number[];
  layout?: LayoutConfig;
}

export interface View {
  id: number;
  database_id: number;
  name: string;
  type: ViewType;
  config: ViewConfig;
  position: string;
  created_at: string;
  updated_at: string;
}

// API response types

export interface DatabaseGetResponse {
  database: Database;
  properties: Property[];
  views: View[];
  records: Record[];
}

export interface RecordsListResponse {
  records: Record[];
  total: number;
  page: number;
  limit: number;
}
