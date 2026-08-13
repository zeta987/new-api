package model

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
)

func IsChannelEnabledForGroupModel(group string, modelName string, channelID int) bool {
	if group == "" || modelName == "" || channelID <= 0 {
		return false
	}
	if !common.MemoryCacheEnabled {
		return isChannelEnabledForGroupModelDB(group, modelName, channelID)
	}

	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	if group2model2channels == nil {
		return false
	}
	if requiresZhipuV4Channel(modelName) {
		channel, ok := channelsIDM[channelID]
		if !ok || channel.Type != constant.ChannelTypeZhipu_v4 {
			return false
		}
	}

	for _, candidate := range ModelMatchCandidates(modelName) {
		if isChannelIDInList(group2model2channels[group][candidate], channelID) {
			return true
		}
	}
	return false
}

func IsChannelEnabledForAnyGroupModel(groups []string, modelName string, channelID int) bool {
	if len(groups) == 0 {
		return false
	}
	for _, g := range groups {
		if IsChannelEnabledForGroupModel(g, modelName, channelID) {
			return true
		}
	}
	return false
}

func isChannelEnabledForGroupModelDB(group string, modelName string, channelID int) bool {
	var count int64
	query := DB.Model(&Ability{}).
		Where(commonGroupCol+" = ? and model IN ? and channel_id = ? and enabled = ?", group, ModelMatchCandidates(modelName), channelID, true).
		Count(&count)
	if requiresZhipuV4Channel(modelName) {
		query = DB.Model(&Ability{}).
			Joins("JOIN channels ON channels.id = abilities.channel_id").
			Where("abilities."+commonGroupCol+" = ? and abilities.model IN ? and abilities.channel_id = ? and abilities.enabled = ? and channels.type = ?", group, ModelMatchCandidates(modelName), channelID, true, constant.ChannelTypeZhipu_v4).
			Count(&count)
	}
	err := query.Error
	return err == nil && count > 0
}

func isChannelIDInList(list []int, channelID int) bool {
	for _, id := range list {
		if id == channelID {
			return true
		}
	}
	return false
}
