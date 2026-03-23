package protocol

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ConversationIdentityResolver 会话身份解析器
type ConversationIdentityResolver interface {
	// Resolve 通过观测引用解析逻辑身份
	Resolve(ref ConversationRef) (*ConversationIdentity, float64, error)

	// GenerateIdentityHash 生成身份哈希（基于多维度特征）
	GenerateIdentityHash(ref ConversationRef) string

	// MatchFeatures 匹配特征（用于冲突检测）
	MatchFeatures(ref ConversationRef, identity *ConversationIdentity) float64

	// CreateIdentity 创建新逻辑身份
	CreateIdentity(ref ConversationRef) (*ConversationIdentity, error)

	// UpdateIdentity 更新逻辑身份
	UpdateIdentity(identity *ConversationIdentity, ref ConversationRef) error
}

// DefaultIdentityResolver 默认实现
type DefaultIdentityResolver struct{}

func (r *DefaultIdentityResolver) GenerateIdentityHash(ref ConversationRef) string {
	// 基于多维度特征生成哈希（不使用 HostWindowHandle）
	// 特征优先级：
	// 1. app instance id + display name + preview text
	// 2. app instance id + avatar hash
	// 3. app instance id + list position neighborhood
	// 4. app instance id + recent message fingerprint

	features := []string{
		ref.AppInstance.InstanceID,
		ref.DisplayName,
		ref.PreviewText,
		ref.AvatarHash,
	}

	data := ""
	for _, f := range features {
		if f != "" {
			data += f + "|"
		}
	}

	// 补充 list position neighborhood（相对位置）
	if ref.ListPosition >= 0 {
		data += fmt.Sprintf("pos:%d|", ref.ListPosition)
	}

	// 补充 list neighborhood hint（邻域提示列表）
	// 标准化处理：去空、trim、固定顺序（排序）
	if len(ref.ListNeighborhoodHint) > 0 {
		normalizedHints := []string{}
		for _, hint := range ref.ListNeighborhoodHint {
			trimmed := strings.TrimSpace(hint)
			if trimmed != "" {
				normalizedHints = append(normalizedHints, trimmed)
			}
		}
		// 排序确保顺序变化不影响 hash
		sort.Strings(normalizedHints)
		for _, hint := range normalizedHints {
			data += fmt.Sprintf("hint:%s|", hint)
		}
	}

	// 补充 recent message fingerprint（最近消息指纹）
	// 当指纹存在时，它会影响 identity hash
	// 如果指纹变化，identity hash 也会变化
	if ref.RecentMessageFingerprint != "" {
		data += fmt.Sprintf("recent_fp:%s|", ref.RecentMessageFingerprint)
	}

	if data == "" {
		// fallback: 使用应用 ID + 时间戳
		data = ref.AppInstance.AppID + "|" + time.Now().Format("20060102")
	}

	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

func (r *DefaultIdentityResolver) Resolve(ref ConversationRef) (*ConversationIdentity, float64, error) {
	// TODO: 实现解析逻辑
	// 1. 生成身份哈希
	hash := r.GenerateIdentityHash(ref)
	// 2. 查询数据库或缓存
	// 3. 返回匹配的身份和置信度
	identity := &ConversationIdentity{
		IdentityHash: hash,
		DisplayName:  ref.DisplayName,
		AvatarHash:   ref.AvatarHash,
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	return identity, 0.8, nil
}

func (r *DefaultIdentityResolver) MatchFeatures(ref ConversationRef, identity *ConversationIdentity) float64 {
	// 基于多维度特征匹配
	score := 0.0

	// 1. 显示名称匹配
	if ref.DisplayName == identity.DisplayName {
		score += 0.3
	}

	// 2. 头像哈希匹配
	if ref.AvatarHash != "" && ref.AvatarHash == identity.AvatarHash {
		score += 0.3
	}

	// 3. 预览文本匹配（最近消息）
	if ref.PreviewText != "" {
		score += 0.2
	}

	// 4. 列表位置稳定性（如果位置接近）
	if ref.ListPosition >= 0 {
		score += 0.2
	}

	return score
}

func (r *DefaultIdentityResolver) CreateIdentity(ref ConversationRef) (*ConversationIdentity, error) {
	hash := r.GenerateIdentityHash(ref)
	identity := &ConversationIdentity{
		IdentityHash: hash,
		DisplayName:  ref.DisplayName,
		AvatarHash:   ref.AvatarHash,
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	return identity, nil
}

func (r *DefaultIdentityResolver) UpdateIdentity(identity *ConversationIdentity, ref ConversationRef) error {
	identity.DisplayName = ref.DisplayName
	identity.AvatarHash = ref.AvatarHash
	identity.UpdatedAt = time.Now()
	return nil
}
