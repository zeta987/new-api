package helper

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/reasoning"
	"github.com/gin-gonic/gin"
)

func ModelMappedHelper(c *gin.Context, info *common.RelayInfo, request dto.Request) error {
	if info.ChannelMeta == nil {
		info.ChannelMeta = &common.ChannelMeta{}
	}

	modelMap, err := channelModelMapping(c)
	if err != nil {
		return err
	}

	sourceModel := info.OriginModelName
	if base, effort, ok := glmEffortAliasForRedirect(info); ok {
		info.ModelSuffixReasoningEffort = effort
		info.SetReasoningEffort(effort)
		// An explicit redirect for the alias itself stays authoritative; without
		// one the redirect chain resolves from the base model, because channel
		// selection and pricing already treat the alias as that base model.
		if _, exists := modelMap[info.OriginModelName]; !exists {
			sourceModel = base
			info.UpstreamModelName = base
			info.IsModelMapped = true
		}
	}

	if len(modelMap) > 0 {
		// 支持链式模型重定向，最终使用链尾的模型
		currentModel := sourceModel
		visitedModels := map[string]bool{
			currentModel: true,
		}
		for {
			mappedModel, exists := modelMap[currentModel]
			if !exists || mappedModel == "" {
				break
			}
			// 模型重定向循环检测，避免无限循环
			if visitedModels[mappedModel] {
				if mappedModel != currentModel {
					return errors.New("model_mapping_contains_cycle")
				}
				if currentModel != sourceModel {
					info.IsModelMapped = true
				}
				break
			}
			visitedModels[mappedModel] = true
			currentModel = mappedModel
			info.IsModelMapped = true
		}
		if info.IsModelMapped {
			info.UpstreamModelName = currentModel
		}
	}

	if request != nil {
		request.SetModelName(info.UpstreamModelName)
	}
	return nil
}

func channelModelMapping(c *gin.Context) (map[string]string, error) {
	modelMapping := c.GetString("model_mapping")
	if modelMapping == "" || modelMapping == "{}" {
		return nil, nil
	}
	modelMap := make(map[string]string)
	if err := json.Unmarshal([]byte(modelMapping), &modelMap); err != nil {
		return nil, fmt.Errorf("unmarshal_model_mapping_failed")
	}
	return modelMap, nil
}

// glmEffortAliasForRedirect splits a validated GLM reasoning effort alias for
// channel types whose OpenAI-compatible adaptor applies the effort as a request
// field. Zhipu V4 is excluded because its own adaptor already resolves the alias
// against the upstream model name.
func glmEffortAliasForRedirect(info *common.RelayInfo) (baseModel string, effort string, ok bool) {
	if info.ChannelType == constant.ChannelTypeZhipu_v4 || !constant.SupportsGLMReasoningEffortAlias(info.ChannelType) {
		return "", "", false
	}
	return reasoning.ParseGLMReasoningEffortSuffix(info.OriginModelName)
}
