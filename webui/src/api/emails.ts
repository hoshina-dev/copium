import { http } from "./http";
import type { OutboxRow, SendEmailRequest, SendEmailResponse, UUID } from "./types";

const base = "/api/v1/emails";

export const emailsApi = {
  send: (req: SendEmailRequest) => http.post<SendEmailResponse>(`${base}/send`, req),
  getOutbox: (id: UUID) => http.get<OutboxRow>(`${base}/${id}`),
};
