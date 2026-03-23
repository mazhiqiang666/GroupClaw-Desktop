package unit

import (
	"testing"

	"github.com/yourorg/auto-customer-service/pkg/protocol"
)

func TestConversationIdentityResolver(t *testing.T) {
	resolver := &protocol.DefaultIdentityResolver{}

	// 构造观测引用
	ref := protocol.ConversationRef{
		DisplayName: "张三",
		PreviewText: "你好",
		AvatarHash:  "abc123",
		ListPosition: 0,
		AppInstance: protocol.AppInstanceRef{
			AppID:      "wechat",
			InstanceID: "instance_001",
		},
	}

	// 生成身份哈希
	hash := resolver.GenerateIdentityHash(ref)

	// 验证哈希不为空
	if hash == "" {
		t.Error("Identity hash should not be empty")
	}

	// 验证哈希长度（SHA256 为 64 字符）
	if len(hash) != 64 {
		t.Errorf("Identity hash length mismatch: got %d, want 64", len(hash))
	}

	// 验证哈希不依赖 HostWindowHandle
	// 构造两个 ConversationRef，唯一差异只在 HostWindowHandle
	ref1 := protocol.ConversationRef{
		HostWindowHandle: 12345,
		DisplayName:      "张三",
		PreviewText:      "你好",
		AvatarHash:       "abc123",
		ListPosition:     0,
		AppInstance: protocol.AppInstanceRef{
			AppID:      "wechat",
			InstanceID: "instance_001",
		},
	}

	ref2 := protocol.ConversationRef{
		HostWindowHandle: 99999, // 不同的窗口句柄
		DisplayName:      "张三",
		PreviewText:      "你好",
		AvatarHash:       "abc123",
		ListPosition:     0,
		AppInstance: protocol.AppInstanceRef{
			AppID:      "wechat",
			InstanceID: "instance_001",
		},
	}

	hash1 := resolver.GenerateIdentityHash(ref1)
	hash2 := resolver.GenerateIdentityHash(ref2)

	// 断言两个哈希相同，证明 identity hash 不依赖 HostWindowHandle
	if hash1 != hash2 {
		t.Errorf("Identity hash should not depend on HostWindowHandle: hash1=%s, hash2=%s", hash1, hash2)
	}
}

func TestConversationIdentityResolver_MatchFeatures(t *testing.T) {
	resolver := &protocol.DefaultIdentityResolver{}

	// 构造观测引用
	ref := protocol.ConversationRef{
		DisplayName:  "张三",
		PreviewText:  "你好",
		AvatarHash:   "abc123",
		ListPosition: 0,
		AppInstance: protocol.AppInstanceRef{
			AppID:      "wechat",
			InstanceID: "instance_001",
		},
	}

	// 创建身份
	identity, err := resolver.CreateIdentity(ref)
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}

	// 测试匹配
	score := resolver.MatchFeatures(ref, identity)

	// 验证匹配分数
	if score <= 0 {
		t.Errorf("MatchFeatures score should be > 0, got %f", score)
	}

	// 测试不匹配的情况
	ref2 := protocol.ConversationRef{
		DisplayName:  "李四", // 不同的名称
		PreviewText:  "你好",
		AvatarHash:   "abc123",
		ListPosition: 0,
		AppInstance: protocol.AppInstanceRef{
			AppID:      "wechat",
			InstanceID: "instance_001",
		},
	}

	score2 := resolver.MatchFeatures(ref2, identity)

	// 验证不匹配的分数较低
	if score2 >= score {
		t.Errorf("MatchFeatures score for different name should be lower: got %f, want < %f", score2, score)
	}
}

// TestConversationIdentityResolver_ListNeighborhoodHintOrder 测试 ListNeighborhoodHint 顺序变化不影响 hash
func TestConversationIdentityResolver_ListNeighborhoodHintOrder(t *testing.T) {
	resolver := &protocol.DefaultIdentityResolver{}

	// 构造两个 ConversationRef，唯一差异是 ListNeighborhoodHint 的顺序
	ref1 := protocol.ConversationRef{
		DisplayName:      "张三",
		PreviewText:      "你好",
		AvatarHash:       "abc123",
		ListPosition:     0,
		ListNeighborhoodHint: []string{"hint1", "hint2", "hint3"},
		AppInstance: protocol.AppInstanceRef{
			AppID:      "wechat",
			InstanceID: "instance_001",
		},
	}

	ref2 := protocol.ConversationRef{
		DisplayName:      "张三",
		PreviewText:      "你好",
		AvatarHash:       "abc123",
		ListPosition:     0,
		ListNeighborhoodHint: []string{"hint3", "hint1", "hint2"}, // 顺序不同
		AppInstance: protocol.AppInstanceRef{
			AppID:      "wechat",
			InstanceID: "instance_001",
		},
	}

	hash1 := resolver.GenerateIdentityHash(ref1)
	hash2 := resolver.GenerateIdentityHash(ref2)

	// 断言两个哈希相同，证明 ListNeighborhoodHint 顺序不影响 hash
	if hash1 != hash2 {
		t.Errorf("Identity hash should not depend on ListNeighborhoodHint order: hash1=%s, hash2=%s", hash1, hash2)
	}
}

// TestConversationIdentityResolver_ListNeighborhoodHintNormalization 测试 ListNeighborhoodHint 标准化处理
func TestConversationIdentityResolver_ListNeighborhoodHintNormalization(t *testing.T) {
	resolver := &protocol.DefaultIdentityResolver{}

	// 构造两个 ConversationRef，唯一差异是 ListNeighborhoodHint 的空白字符
	ref1 := protocol.ConversationRef{
		DisplayName:      "张三",
		ListNeighborhoodHint: []string{"  hint1  ", "hint2", "  hint3  "},
		AppInstance: protocol.AppInstanceRef{
			AppID:      "wechat",
			InstanceID: "instance_001",
		},
	}

	ref2 := protocol.ConversationRef{
		DisplayName:      "张三",
		ListNeighborhoodHint: []string{"hint1", "hint2", "hint3"}, // 去掉空白
		AppInstance: protocol.AppInstanceRef{
			AppID:      "wechat",
			InstanceID: "instance_001",
		},
	}

	hash1 := resolver.GenerateIdentityHash(ref1)
	hash2 := resolver.GenerateIdentityHash(ref2)

	// 断言两个哈希相同，证明空白字符被标准化
	if hash1 != hash2 {
		t.Errorf("Identity hash should be same after normalization: hash1=%s, hash2=%s", hash1, hash2)
	}
}

// TestConversationIdentityResolver_ListNeighborhoodHintEmpty 测试 ListNeighborhoodHint 空字符串过滤
func TestConversationIdentityResolver_ListNeighborhoodHintEmpty(t *testing.T) {
	resolver := &protocol.DefaultIdentityResolver{}

	// 构造两个 ConversationRef，唯一差异是 ListNeighborhoodHint 包含空字符串
	ref1 := protocol.ConversationRef{
		DisplayName:      "张三",
		ListNeighborhoodHint: []string{"hint1", "", "hint2"},
		AppInstance: protocol.AppInstanceRef{
			AppID:      "wechat",
			InstanceID: "instance_001",
		},
	}

	ref2 := protocol.ConversationRef{
		DisplayName:      "张三",
		ListNeighborhoodHint: []string{"hint1", "hint2"}, // 没有空字符串
		AppInstance: protocol.AppInstanceRef{
			AppID:      "wechat",
			InstanceID: "instance_001",
		},
	}

	hash1 := resolver.GenerateIdentityHash(ref1)
	hash2 := resolver.GenerateIdentityHash(ref2)

	// 断言两个哈希相同，证明空字符串被过滤
	if hash1 != hash2 {
		t.Errorf("Identity hash should be same after filtering empty hints: hash1=%s, hash2=%s", hash1, hash2)
	}
}

// TestConversationIdentityResolver_RecentMessageFingerprint 测试 RecentMessageFingerprint 变化影响 hash
func TestConversationIdentityResolver_RecentMessageFingerprint(t *testing.T) {
	resolver := &protocol.DefaultIdentityResolver{}

	// 构造两个 ConversationRef，唯一差异是 RecentMessageFingerprint
	ref1 := protocol.ConversationRef{
		DisplayName:             "张三",
		RecentMessageFingerprint: "fp123",
		AppInstance: protocol.AppInstanceRef{
			AppID:      "wechat",
			InstanceID: "instance_001",
		},
	}

	ref2 := protocol.ConversationRef{
		DisplayName:             "张三",
		RecentMessageFingerprint: "fp456", // 不同的指纹
		AppInstance: protocol.AppInstanceRef{
			AppID:      "wechat",
			InstanceID: "instance_001",
		},
	}

	hash1 := resolver.GenerateIdentityHash(ref1)
	hash2 := resolver.GenerateIdentityHash(ref2)

	// 断言两个哈希不同，证明 RecentMessageFingerprint 影响 hash
	if hash1 == hash2 {
		t.Errorf("Identity hash should change when RecentMessageFingerprint changes: hash1=%s, hash2=%s", hash1, hash2)
	}
}

// TestConversationIdentityResolver_RecentMessageFingerprintStability 测试 RecentMessageFingerprint 稳定性
func TestConversationIdentityResolver_RecentMessageFingerprintStability(t *testing.T) {
	resolver := &protocol.DefaultIdentityResolver{}

	// 构造相同的 ConversationRef 两次
	ref := protocol.ConversationRef{
		DisplayName:             "张三",
		RecentMessageFingerprint: "fp123",
		AppInstance: protocol.AppInstanceRef{
			AppID:      "wechat",
			InstanceID: "instance_001",
		},
	}

	hash1 := resolver.GenerateIdentityHash(ref)
	hash2 := resolver.GenerateIdentityHash(ref)

	// 断言两个哈希相同，证明相同输入产生相同输出
	if hash1 != hash2 {
		t.Errorf("Identity hash should be stable for same input: hash1=%s, hash2=%s", hash1, hash2)
	}
}
