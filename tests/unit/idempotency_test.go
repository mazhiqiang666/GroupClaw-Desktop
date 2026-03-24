package unit

import (
	"testing"

	"github.com/mazhiqiang666/GroupClaw-Desktop/internal/agent/idempotency"
)

func TestIdempotencyStore(t *testing.T) {
	store := idempotency.NewMemoryStore()

	// 创建记录
	record := idempotency.Record{
		TaskID:       "task_001",
		DedupeKey:    "dedupe_001",
		Conversation: "conv_001",
		Content:      "测试消息",
		Status:       "pending",
	}

	err := store.CreateRecord(record)
	if err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}

	// 获取记录
	retrieved, err := store.GetRecord("task_001")
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}

	if retrieved.TaskID != "task_001" {
		t.Errorf("TaskID mismatch: got %v, want %v", retrieved.TaskID, "task_001")
	}

	// 检查重复
	duplicate, err := store.CheckDuplicate("dedupe_001")
	if err != nil {
		t.Fatalf("CheckDuplicate failed: %v", err)
	}

	if duplicate == nil {
		t.Error("Duplicate record should not be nil")
	}

	if duplicate.TaskID != "task_001" {
		t.Errorf("Duplicate TaskID mismatch: got %v, want %v", duplicate.TaskID, "task_001")
	}
}

func TestIdempotencyStore_UpdateRecord(t *testing.T) {
	store := idempotency.NewMemoryStore()

	// 创建记录
	record := idempotency.Record{
		TaskID:       "task_002",
		DedupeKey:    "dedupe_002",
		Conversation: "conv_001",
		Content:      "测试消息",
		Status:       "pending",
	}

	err := store.CreateRecord(record)
	if err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}

	// 更新记录
	err = store.UpdateRecord("task_002", idempotency.NewRecordUpdate().
		WithStatus("sending").
		WithMessageID("msg_002").
		WithFingerprint("fp_xyz789").
		WithVerifyCount(3))
	if err != nil {
		t.Fatalf("UpdateRecord failed: %v", err)
	}

	// 获取更新后的记录
	retrieved, err := store.GetRecord("task_002")
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}

	if retrieved.Status != "sending" {
		t.Errorf("Status mismatch: got %v, want %v", retrieved.Status, "sending")
	}

	if retrieved.MessageID != "msg_002" {
		t.Errorf("MessageID mismatch: got %v, want %v", retrieved.MessageID, "msg_002")
	}

	if retrieved.Fingerprint != "fp_xyz789" {
		t.Errorf("Fingerprint mismatch: got %v, want %v", retrieved.Fingerprint, "fp_xyz789")
	}

	if retrieved.VerifyCount != 3 {
		t.Errorf("VerifyCount mismatch: got %v, want %v", retrieved.VerifyCount, 3)
	}
}
