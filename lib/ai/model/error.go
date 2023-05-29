/*
Copyright 2023 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package model

import (
	"encoding/json"
	"fmt"
	"strings"
)

type invalidOutputError struct {
	coarse string
	detail string
}

func newInvalidOutputErrorWithParseError(err error) *invalidOutputError {
	return &invalidOutputError{
		coarse: "json parse error",
		detail: err.Error(),
	}
}

func newInvalidOutputError(coarse string, detail string) *invalidOutputError {
	return &invalidOutputError{
		coarse: coarse,
		detail: detail,
	}
}

func (o *invalidOutputError) Error() string {
	return fmt.Sprintf("%v: %v", o.coarse, o.detail)
}

func parseConversationOutput(text string) (*AgentAction, *AgentFinish, error) {
	cleaned := strings.TrimSpace(text)
	if strings.Contains(cleaned, "```json") {
		cleaned = strings.Split(cleaned, "```json")[1]
	}
	if strings.Contains(cleaned, "```") {
		cleaned = strings.Split(cleaned, "```")[0]
	}
	if strings.HasPrefix(cleaned, "```json") {
		cleaned = cleaned[len("```json"):]
	}
	if strings.HasPrefix(cleaned, "```") {
		cleaned = cleaned[len("```"):]
	}
	if strings.HasSuffix(cleaned, "```") {
		cleaned = cleaned[:len("```")]
	}
	cleaned = strings.TrimSpace(cleaned)
	var response map[string]string
	err := json.Unmarshal([]byte(cleaned), &response)
	if err != nil {
		return nil, nil, newInvalidOutputErrorWithParseError(err)
	}

	action, input := response["action"], response["action_input"]
	if action == actionFinalAnswer {
		return nil, &AgentFinish{output: input}, nil
	}

	return &AgentAction{action: action, input: input}, nil, nil
}
