// Mirrors the Go DTOs in internal/models/dto.go. Keep in sync by hand;
// the canonical source of truth is the OpenAPI spec at /swagger/doc.json.

export type UUID = string;
export type ISODateTime = string;
export type JSONValue = string | number | boolean | null | JSONValue[] | { [k: string]: JSONValue };
export type JSONObject = Record<string, JSONValue>;

export interface Template {
  id: UUID;
  code: string;
  name: string;
  description?: string;
  active_version_id?: UUID;
  created_at: ISODateTime;
  updated_at: ISODateTime;
}

export interface TemplateVersion {
  id: UUID;
  template_id: UUID;
  version: number;
  subject: string;
  body_html: string;
  body_text?: string;
  params_schema: JSONObject;
  from_address?: string;
  created_by?: string;
  created_at: ISODateTime;
}

export interface CreateTemplateRequest {
  code: string;
  name: string;
  description?: string;
}

export interface CreateTemplateVersionRequest {
  subject: string;
  body_html: string;
  body_text?: string;
  params_schema: JSONObject;
  from_address?: string;
}

export interface SendEmailRequest {
  template_id: UUID;
  // Provide EXACTLY one of user_id (resolved via custapi) or to_address
  // (direct dispatch for external addresses not in our system).
  user_id?: UUID;
  to_address?: string;
  params: JSONObject;
}

export interface SendEmailResponse {
  outbox_id: UUID;
  status: string;
}

export type OutboxStatus =
  | "queued"
  | "sending"
  | "sent"
  | "failed"
  | "dead";

export interface OutboxRow {
  id: UUID;
  template_version_id: UUID;
  user_id?: UUID;
  to_address: string;
  subject: string;
  status: OutboxStatus;
  attempts: number;
  max_attempts: number;
  scheduled_at: ISODateTime;
  last_error?: string;
  provider?: string;
  provider_message_id?: string;
  sent_at?: ISODateTime;
  created_at: ISODateTime;
  updated_at: ISODateTime;
}

export interface ApiError {
  error: string;
}
