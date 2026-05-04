import { http } from "./http";
import type {
  CreateTemplateRequest,
  CreateTemplateVersionRequest,
  Template,
  TemplateVersion,
  UUID,
} from "./types";

const base = "/api/v1/templates";

export const templatesApi = {
  list: () => http.get<Template[]>(`${base}/`),
  get: (id: UUID) => http.get<Template>(`${base}/${id}`),
  create: (req: CreateTemplateRequest) => http.post<Template>(`${base}/`, req),

  listVersions: (id: UUID) => http.get<TemplateVersion[]>(`${base}/${id}/versions`),
  getVersion: (id: UUID, version: number) =>
    http.get<TemplateVersion>(`${base}/${id}/versions/${version}`),
  createVersion: (id: UUID, req: CreateTemplateVersionRequest) =>
    http.post<TemplateVersion>(`${base}/${id}/versions`, req),

  setActive: (id: UUID, versionId: UUID) =>
    http.patch<void>(`${base}/${id}/active-version`, { version_id: versionId }),
};
