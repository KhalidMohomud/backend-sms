package service

import (
	"backendapi/internal/model"
	"backendapi/internal/repository"
	"context"
	"encoding/json"
	"fmt"
)

type RequestMetadata struct {
	IPAddress string
	UserAgent string
}

type AuditWriter struct {
	repository repository.AuditRepository
}

func NewAuditWriter(repository repository.AuditRepository) *AuditWriter {
	return &AuditWriter{repository: repository}
}

func (w *AuditWriter) Write(
	ctx context.Context,
	userID, schoolID *uint64,
	action, resource string,
	recordID *uint64,
	request RequestMetadata,
	fields map[string]any,
) error {
	if fields == nil {
		fields = make(map[string]any)
	}
	if request.UserAgent != "" {
		fields["user_agent"] = request.UserAgent
	}
	metadata, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("encode audit metadata: %w", err)
	}
	entry := &model.AuditLog{
		UserID:       userID,
		SchoolID:     schoolID,
		Action:       action,
		ResourceType: resource,
		RecordID:     recordID,
		IPAddress:    request.IPAddress,
		Metadata:     metadata,
	}
	if err := w.repository.Create(ctx, entry); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}
