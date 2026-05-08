package internal_custom_llm_callers

import (
	"context"
	"time"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type verifyCredentialCaller struct {
	CustomLLM
}

func NewVerifyCredentialCaller(logger commons.Logger, credential *protos.Credential) internal_callers.Verifier {
	return &verifyCredentialCaller{
		CustomLLM: customLLM(logger, credential),
	}
}

func (stc *verifyCredentialCaller) CredentialVerifier(
	ctx context.Context,
	options *internal_callers.CredentialVerifierOptions) (*string, error) {
	client, err := stc.GetClient()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	if _, err := client.Models.List(ctx); err != nil {
		return nil, err
	}
	return nil, nil
}
