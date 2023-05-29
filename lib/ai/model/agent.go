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
	"context"
	"encoding/json"
	"strings"

	"github.com/gravitational/trace"
	"github.com/sashabaranov/go-openai"
)

type Agent struct {
	systemMessage string
	tools         []Tool
	LLMPrefix     string
}

type AgentAction struct {
	action      string
	actionInput string
	log         string
}

type AgentFinish struct {
	output string
}

type executionState struct {
	llm               openai.Client
	chatHistory       []openai.ChatCompletionMessage
	humanMessage      openai.ChatCompletionMessage
	intermediateSteps []AgentAction
	observations      []string
}

func (a *Agent) plan(ctx context.Context, state *executionState) (*AgentAction, *AgentFinish, error) {
	scratchpad := a.constructScratchpad(state.intermediateSteps, state.observations)
	prompt := a.createPrompt(state.chatHistory, scratchpad, state.humanMessage)
	resp, err := state.llm.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    openai.GPT4,
			Messages: prompt,
		},
	)
	if err != nil {
		return nil, nil, trace.Wrap(err)
	}

	llmOut := resp.Choices[0].Message.Content
	return parseConversationOutput(llmOut)
}

func (a *Agent) createPrompt(chatHistory, agentScratchpad []openai.ChatCompletionMessage, humanMessage openai.ChatCompletionMessage) []openai.ChatCompletionMessage {
	prompt := make([]openai.ChatCompletionMessage, 0)
	prompt = append(prompt, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: a.systemMessage,
	})
	prompt = append(prompt, chatHistory...)

	toolList := strings.Builder{}
	toolNames := make([]string, 0, len(a.tools))
	for _, tool := range a.tools {
		toolNames = append(toolNames, tool.Name())
		toolList.WriteString("> ")
		toolList.WriteString(tool.Name())
		toolList.WriteString(": ")
		toolList.WriteString(tool.Description())
		toolList.WriteString("\n")
	}

	formatInstructions := conversationParserFormatInstructionsPrompt(toolNames)
	newHumanMessage := conversationToolUsePrompt(toolList.String(), formatInstructions, humanMessage.Content)
	prompt = append(prompt, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: newHumanMessage,
	})

	prompt = append(prompt, agentScratchpad...)
	return prompt
}

func (a *Agent) constructScratchpad(intermediateSteps []AgentAction, observations []string) []openai.ChatCompletionMessage {
	var thoughts []openai.ChatCompletionMessage
	for i, action := range intermediateSteps {
		thoughts = append(thoughts, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: action.log,
		}, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: conversationToolResponse(observations[i]),
		})
	}

	return thoughts
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
		return nil, nil, trace.Wrap(err)
	}

	action, actionInput := response["action"], response["action_input"]
	if action == "Final Answer" {
		return nil, &AgentFinish{output: actionInput}, nil
	}

	return &AgentAction{action: action, actionInput: actionInput}, nil, nil
}
