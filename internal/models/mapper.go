package models

func TemplateToResponse(t *EmailTemplate) TemplateResponse {
	return TemplateResponse{
		ID:              t.ID,
		Code:            t.Code,
		Name:            t.Name,
		Description:     t.Description,
		ActiveVersionID: t.ActiveVersionID,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
	}
}

func TemplateVersionToResponse(v *EmailTemplateVersion) TemplateVersionResponse {
	return TemplateVersionResponse{
		ID:           v.ID,
		TemplateID:   v.TemplateID,
		Version:      v.Version,
		Subject:      v.Subject,
		BodyHTML:     v.BodyHTML,
		BodyText:     v.BodyText,
		ParamsSchema: v.ParamsSchema,
		FromAddress:  v.FromAddress,
		CreatedBy:    v.CreatedBy,
		CreatedAt:    v.CreatedAt,
	}
}

func OutboxToResponse(o *EmailOutbox) OutboxResponse {
	return OutboxResponse{
		ID:                o.ID,
		TemplateVersionID: o.TemplateVersionID,
		UserID:            o.UserID,
		ToAddress:         o.ToAddress,
		Subject:           o.Subject,
		Status:            string(o.Status),
		Attempts:          o.Attempts,
		MaxAttempts:       o.MaxAttempts,
		ScheduledAt:       o.ScheduledAt,
		LastError:         o.LastError,
		Provider:          o.Provider,
		ProviderMessageID: o.ProviderMessageID,
		SentAt:            o.SentAt,
		CreatedAt:         o.CreatedAt,
		UpdatedAt:         o.UpdatedAt,
	}
}
