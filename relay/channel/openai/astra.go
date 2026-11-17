package openai

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/reasoning"
)

func validateRequiredReasoning(model string, config *dto.Reasoning) error {
	if config == nil {
		return nil
	}
	if config.Effort != "" && !reasoning.IsGPT6AstraEffort(config.Effort) {
		return fmt.Errorf("%s reasoning effort must be low, medium, high, xhigh, or max", model)
	}
	if len(config.Mode) == 0 || string(config.Mode) == "null" {
		return nil
	}
	var mode string
	if err := common.Unmarshal(config.Mode, &mode); err != nil || mode != "standard" && mode != "pro" {
		return fmt.Errorf("%s reasoning mode must be standard or pro", model)
	}
	return nil
}

func normalizeRequiredReasoningResponses(request *dto.OpenAIResponsesRequest, capabilityModel string) error {
	if !reasoning.IsGPT6RequiredReasoningModel(capabilityModel) {
		return nil
	}
	if err := validateRequiredReasoning(capabilityModel, request.Reasoning); err != nil {
		return err
	}
	request.Temperature = nil
	request.TopP = nil
	request.TopLogProbs = nil
	if len(request.Include) > 0 && string(request.Include) != "null" {
		var includes []string
		if err := common.Unmarshal(request.Include, &includes); err != nil {
			return fmt.Errorf("%s include must be an array of strings: %w", capabilityModel, err)
		}
		filtered := make([]string, 0, len(includes))
		for _, include := range includes {
			if include != "message.output_text.logprobs" {
				filtered = append(filtered, include)
			}
		}
		encoded, err := common.Marshal(filtered)
		if err != nil {
			return err
		}
		request.Include = encoded
	}
	return nil
}

// Preserve the client's reasoning requirements when mapping to a vendor ID.
func openAIReasoningCapabilityModel(info *relaycommon.RelayInfo, model string) string {
	if reasoning.IsGPT6RequiredReasoningModel(model) || info == nil {
		return model
	}
	origin := reasoning.ParseModelModifiers(info.OriginModelName).Base
	if reasoning.IsGPT6RequiredReasoningModel(origin) {
		return origin
	}
	return model
}
