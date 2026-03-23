package idempotency

import (
	"errors"
	"sync"
	"time"
)

// MemoryStore 内存实现的幂等存储
type MemoryStore struct {
	records map[string]*Record
	mu      sync.RWMutex
}

// NewMemoryStore 创建内存存储实例
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		records: make(map[string]*Record),
	}
}

// CreateRecord 创建记录
func (s *MemoryStore) CreateRecord(record Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查 TaskID 是否已存在
	if _, exists := s.records[record.TaskID]; exists {
		return errors.New("record already exists")
	}

	// 检查 DedupeKey 是否已存在
	for _, r := range s.records {
		if r.DedupeKey == record.DedupeKey {
			return errors.New("dedupe_key already exists")
		}
	}

	record.CreatedAt = time.Now()
	record.UpdatedAt = time.Now()
	s.records[record.TaskID] = &record
	return nil
}

// GetRecord 获取记录
func (s *MemoryStore) GetRecord(taskID string) (*Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, exists := s.records[taskID]
	if !exists {
		return nil, errors.New("record not found")
	}

	// 返回副本避免外部修改
	copy := *record
	return &copy, nil
}

// UpdateRecord 更新记录（使用强类型更新参数）
func (s *MemoryStore) UpdateRecord(taskID string, updates RecordUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.records[taskID]
	if !exists {
		return errors.New("record not found")
	}

	// 应用更新（使用强类型参数，无需类型断言）
	if updates.Status != nil {
		record.Status = *updates.Status
	}
	if updates.MessageID != nil {
		record.MessageID = *updates.MessageID
	}
	if updates.Fingerprint != nil {
		record.Fingerprint = *updates.Fingerprint
	}
	if updates.VerifyStatus != nil {
		record.VerifyStatus = *updates.VerifyStatus
	}
	if updates.VerifyCount != nil {
		record.VerifyCount = *updates.VerifyCount
	}

	record.UpdatedAt = time.Now()
	return nil
}

// CheckDuplicate 检查重复
func (s *MemoryStore) CheckDuplicate(dedupeKey string) (*Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, record := range s.records {
		if record.DedupeKey == dedupeKey {
			copy := *record
			return &copy, nil
		}
	}

	return nil, nil
}

// ListRecords 列出会话记录
func (s *MemoryStore) ListRecords(conversationID string) ([]Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var records []Record
	for _, record := range s.records {
		if record.Conversation == conversationID {
			records = append(records, *record)
		}
	}

	return records, nil
}
