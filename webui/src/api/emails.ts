import { http } from "./http";
import type { OutboxRow, SendEmailRequest, SendEmailResponse, UUID } from "./types";

const base = "/api/v1/emails";

export interface ListOutboxQuery {
  status?: string;
  from?: string; // RFC3339
  to?: string; // RFC3339
  limit?: number;
}

function toQuery(q: ListOutboxQuery): string {
  const p = new URLSearchParams();
  if (q.status) p.set("status", q.status);
  if (q.from) p.set("from", q.from);
  if (q.to) p.set("to", q.to);
  if (q.limit) p.set("limit", String(q.limit));
  const s = p.toString();
  return s ? `?${s}` : "";
}

export const emailsApi = {
  send: (req: SendEmailRequest) => http.post<SendEmailResponse>(`${base}/send`, req),
  getOutbox: (id: UUID) => http.get<OutboxRow>(`${base}/${id}`),
  listOutbox: (q: ListOutboxQuery = {}) => http.get<OutboxRow[]>(`${base}/${toQuery(q)}`),
};
