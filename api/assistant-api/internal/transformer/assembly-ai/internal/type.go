// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package assemblyai_internal

type TranscriptMessage struct {
	TurnOrder           int     `json:"turn_order"`
	TurnIsFormatted     bool    `json:"turn_is_formatted"`
	EndOfTurn           bool    `json:"end_of_turn"`
	Transcript          string  `json:"transcript"`
	EndOfTurnConfidence float64 `json:"end_of_turn_confidence"`
	Words               []Word  `json:"words"`
	Type                string  `json:"type"`
	ErrorCode           int     `json:"error_code,omitempty"`
	Error               string  `json:"error,omitempty"`
}

type Word struct {
	Start       int     `json:"start"`
	End         int     `json:"end"`
	Text        string  `json:"text"`
	Confidence  float64 `json:"confidence"`
	WordIsFinal bool    `json:"word_is_final"`
}
