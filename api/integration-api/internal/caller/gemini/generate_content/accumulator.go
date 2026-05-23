// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_gemini_generate_content

import (
	"time"

	"google.golang.org/genai"
)

type googleChatCompletionAccumulator struct {
	Candidates      []*genai.Candidate `json:"candidates,omitempty"`
	candidateStates []candidateStreamState
	justFinished    candidateStreamState

	ResponseID     string                                       `json:"responseId,omitempty"`
	CreateTime     time.Time                                    `json:"createTime,omitempty"`
	ModelVersion   string                                       `json:"modelVersion,omitempty"`
	PromptFeedback *genai.GenerateContentResponsePromptFeedback `json:"promptFeedback,omitempty"`
	UsageMetadata  *genai.GenerateContentResponseUsageMetadata  `json:"usageMetadata,omitempty"`
}

type streamState int

const (
	streamStateEmpty streamState = iota
	streamStateContent
	streamStateFunctionCall
	streamStateFunctionResponse
	streamStateFinished
)

type candidateStreamState struct {
	state streamState
	index int
}

func (acc *googleChatCompletionAccumulator) AddChunk(resp *genai.GenerateContentResponse) bool {
	acc.justFinished = candidateStreamState{}

	if acc.ResponseID == "" {
		acc.ResponseID = resp.ResponseID
		acc.CreateTime = resp.CreateTime
		acc.ModelVersion = resp.ModelVersion
		acc.PromptFeedback = resp.PromptFeedback
		acc.UsageMetadata = resp.UsageMetadata
	}

	for _, incoming := range resp.Candidates {
		index := int(incoming.Index)
		acc.Candidates = expandToFit(acc.Candidates, index)
		acc.candidateStates = expandToFit(acc.candidateStates, index)

		existing := acc.Candidates[index]
		if existing == nil {
			acc.Candidates[index] = &genai.Candidate{}
			existing = acc.Candidates[index]
		}

		if incoming.Content != nil {
			if existing.Content == nil {
				existing.Content = &genai.Content{Role: incoming.Content.Role}
			}
			existing.Content.Parts = append(existing.Content.Parts, incoming.Content.Parts...)
		}

		existing.FinishReason = incoming.FinishReason
		existing.FinishMessage = incoming.FinishMessage
		existing.TokenCount += incoming.TokenCount

		newState := detectState(incoming)
		prevState := acc.candidateStates[index]
		if prevState != newState {
			acc.justFinished = prevState
		}
		acc.candidateStates[index] = newState
	}

	return true
}

func detectState(c *genai.Candidate) candidateStreamState {
	if c.FinishReason != "" {
		return candidateStreamState{state: streamStateFinished, index: int(c.Index)}
	}
	if c.Content != nil && len(c.Content.Parts) > 0 {
		for _, p := range c.Content.Parts {
			if p.FunctionCall != nil {
				return candidateStreamState{state: streamStateFunctionCall, index: int(c.Index)}
			}
			if p.FunctionResponse != nil {
				return candidateStreamState{state: streamStateFunctionResponse, index: int(c.Index)}
			}
			if p.Text != "" {
				return candidateStreamState{state: streamStateContent, index: int(c.Index)}
			}
		}
	}
	return candidateStreamState{state: streamStateEmpty, index: int(c.Index)}
}

func expandToFit[T any](slice []T, index int) []T {
	if index < len(slice) {
		return slice
	}
	newSlice := make([]T, index+1)
	copy(newSlice, slice)
	return newSlice
}
