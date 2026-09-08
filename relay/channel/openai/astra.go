package openai

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/reasoning"
)

func validateAstraReasoning(config *dto.Reasoning) error {
	if config == nil {
		return nil
	}
	if config.Effort != "" && !reasoning.IsGPT6AstraEffort(config.Effort) {
		return fmt.Errorf("gpt-6-astra reasoning effort must be low, medium, high, xhigh, or max")
	}
	if len(config.Mode) == 0 || string(config.Mode) == "null" {
		return nil
	}
	var mode string
	if err := common.Unmarshal(config.Mode, &mode); err != nil || mode != "standard" && mode != "pro" {
		return fmt.Errorf("gpt-6-astra reasoning mode must be standard or pro")
	}
	return nil
}

func normalizeAstraResponses(request *dto.OpenAIResponsesRequest, capabilityModel string) error {
	if !reasoning.IsGPT6AstraModel(capabilityModel) {
		return nil
	}
	if err := validateAstraReasoning(request.Reasoning); err != nil {
		return err
	}
	request.Temperature = nil
	request.TopP = nil
	request.TopLogProbs = nil
	if len(request.Include) > 0 && string(request.Include) != "null" {
		var includes []string
		if err := common.Unmarshal(request.Include, &includes); err != nil {
			return fmt.Errorf("gpt-6-astra include must be an array of strings: %w", err)
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

// Model mapping changes the vendor ID, not the client's Astra contract.
func openAIReasoningCapabilityModel(info *relaycommon.RelayInfo, model string) string {
	if reasoning.IsGPT6AstraModel(model) || info == nil {
		return model
	}
	origin := reasoning.ParseModelModifiers(info.OriginModelName).Base
	if reasoning.IsGPT6AstraModel(origin) {
		return origin
	}
	return model
}
