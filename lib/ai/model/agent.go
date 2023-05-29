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
	"strings"
	"time"

	"github.com/gravitational/trace"
	"github.com/sashabaranov/go-openai"
)

const (
	actionFinalAnswer = "Final Answer"
	actionException   = "_Exception"
	maxIterations     = 15
	maxElapsedTime    = 5 * time.Minute
)

type Agent struct {
	systemMessage string
	tools         []Tool
}

type AgentAction struct {
	action string
	input  string
	log    string
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

func (a *Agent) Think(ctx context.Context, llm openai.Client, chatHistory []openai.ChatCompletionMessage, humanMessage openai.ChatCompletionMessage) (string, error) {
	iterations := 0
	start := time.Now()
	shouldExit := func() bool { return iterations > maxIterations || time.Since(start) > maxElapsedTime }
	state := &executionState{
		llm:               llm,
		chatHistory:       chatHistory,
		humanMessage:      humanMessage,
		intermediateSteps: make([]AgentAction, 0),
		observations:      make([]string, 0),
	}

	for {
		// This is intentionally not context-based, as we want to finish the current step before exiting
		// and the concern is not that we're stuck but that we're taking too long over multiple iterations.
		if shouldExit() {
			return "nil", trace.Errorf("timeout: agent took too long to finish")
		}

		output, err := a.takeNextStep(ctx, state)
		if err != nil {
			return "", trace.Wrap(err)
		}

		if output.finish != nil {
			return output.finish.output, nil
		}

		if output.action != nil {
			state.intermediateSteps = append(state.intermediateSteps, *output.action)
			state.observations = append(state.observations, output.observation)
		}

		iterations++
	}
}

type stepOutput struct {
	finish      *AgentFinish
	action      *AgentAction
	observation string
}

func (a *Agent) takeNextStep(ctx context.Context, state *executionState) (stepOutput, error) {
	_, finish, err := a.plan(ctx, state)
	if err, ok := trace.Unwrap(err).(*invalidOutputError); ok {
		action := &AgentAction{
			action: actionException,
			input:  "Invalid or incomplete response",
			log:    err.Error(),
		}

		// The exception tool is currently a bit special, the observation is always equal to the input.
		// We can expand on this in the future to make it handle errors better.
		return stepOutput{action: action, observation: action.input}, nil
	} else if err != nil {
		return stepOutput{}, trace.Wrap(err)
	}

	// If finish is set, the agent is done and did not call upon any tool.
	if finish != nil {
		return stepOutput{finish: finish}, nil
	}

	return stepOutput{}, trace.NotImplemented("agent picked a tool, this should not happen yet")
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
	action, finish, err := parseConversationOutput(llmOut)
	return action, finish, trace.Wrap(err)
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
