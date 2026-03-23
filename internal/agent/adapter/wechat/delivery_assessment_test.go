package wechat

import (
	"testing"
)

// ==================== DeliveryAssessmentRules Basic Tests ====================

func TestDeliveryAssessmentRules_AssessDeliveryState(t *testing.T) {
	rules := NewDeliveryAssessmentRules()

	// Test verified state
	focusEvidence := FocusVerificationEvidence{
		NodeStillExists:    true,
		NodeHasActiveState: true,
		Confidence:         0.9,
		EvidenceCount:      2,
	}
	messageEvidence := SendVerificationEvidence{
		NewMessageNodes:   1,
		MessageNodeAdded:  true,
		ScreenshotChanged: true,
		Confidence:        0.9,
	}

	assessment := rules.AssessDeliveryState(focusEvidence, messageEvidence)

	if assessment.State != "verified" {
		t.Errorf("Expected state 'verified', got '%s'", assessment.State)
	}
	if assessment.Confidence < 0.8 {
		t.Errorf("Expected confidence >= 0.8, got %f", assessment.Confidence)
	}

	// Test sent_unverified state
	focusEvidence.Confidence = 0.5
	messageEvidence.Confidence = 0.5
	assessment = rules.AssessDeliveryState(focusEvidence, messageEvidence)

	if assessment.State != "sent_unverified" {
		t.Errorf("Expected state 'sent_unverified', got '%s'", assessment.State)
	}

	// Test unknown state
	focusEvidence.Confidence = 0.1
	messageEvidence.Confidence = 0.1
	assessment = rules.AssessDeliveryState(focusEvidence, messageEvidence)

	if assessment.State != "unknown" {
		t.Errorf("Expected state 'unknown', got '%s'", assessment.State)
	}
}

func TestDeliveryAssessmentRules_AssessFocusOnlyState(t *testing.T) {
	rules := NewDeliveryAssessmentRules()

	// Test verified state
	focusEvidence := FocusVerificationEvidence{
		NodeStillExists:    true,
		NodeHasActiveState: true,
		Confidence:         0.9,
		EvidenceCount:      2,
	}

	assessment := rules.AssessFocusOnlyState(focusEvidence)

	if assessment.State != "verified" {
		t.Errorf("Expected state 'verified', got '%s'", assessment.State)
	}
	if assessment.Confidence != 0.9 {
		t.Errorf("Expected confidence 0.9, got %f", assessment.Confidence)
	}

	// Test unknown state
	focusEvidence.Confidence = 0.1
	assessment = rules.AssessFocusOnlyState(focusEvidence)

	if assessment.State != "unknown" {
		t.Errorf("Expected state 'unknown', got '%s'", assessment.State)
	}
}

// ==================== DeliveryAssessmentRules Dirty Data Tests ====================

func TestDeliveryAssessmentRules_DirtyData_InvalidEvidence(t *testing.T) {
	rules := NewDeliveryAssessmentRules()

	tests := []struct {
		name             string
		focusEvidence    FocusVerificationEvidence
		messageEvidence  SendVerificationEvidence
		expectState      string
		expectConfidence float64
	}{
		{
			name: "Zero confidence both",
			focusEvidence: FocusVerificationEvidence{
				NodeStillExists: false,
				Confidence:      0.0,
			},
			messageEvidence: SendVerificationEvidence{
				NewMessageNodes: 0,
				Confidence:      0.0,
			},
			expectState:      "unknown",
			expectConfidence: 0.0,
		},
		{
			name: "Very low confidence",
			focusEvidence: FocusVerificationEvidence{
				NodeStillExists: true,
				Confidence:      0.1,
			},
			messageEvidence: SendVerificationEvidence{
				NewMessageNodes: 1,
				Confidence:      0.1,
			},
			expectState:      "unknown",
			expectConfidence: 0.1,
		},
		{
			name: "Borderline verified (0.79)",
			focusEvidence: FocusVerificationEvidence{
				NodeStillExists: true,
				Confidence:      0.79,
			},
			messageEvidence: SendVerificationEvidence{
				NewMessageNodes: 1,
				Confidence:      0.79,
			},
			expectState:      "sent_unverified",
			expectConfidence: 0.79,
		},
		{
			name: "Exactly at verified threshold (0.8)",
			focusEvidence: FocusVerificationEvidence{
				NodeStillExists: true,
				Confidence:      0.8,
			},
			messageEvidence: SendVerificationEvidence{
				NewMessageNodes: 1,
				Confidence:      0.8,
			},
			expectState:      "verified",
			expectConfidence: 0.8,
		},
		{
			name: "Borderline sent_unverified (0.49)",
			focusEvidence: FocusVerificationEvidence{
				NodeStillExists: true,
				Confidence:      0.49,
			},
			messageEvidence: SendVerificationEvidence{
				NewMessageNodes: 1,
				Confidence:      0.49,
			},
			expectState:      "unknown",
			expectConfidence: 0.49,
		},
		{
			name: "Exactly at sent_unverified threshold (0.5)",
			focusEvidence: FocusVerificationEvidence{
				NodeStillExists: true,
				Confidence:      0.5,
			},
			messageEvidence: SendVerificationEvidence{
				NewMessageNodes: 1,
				Confidence:      0.5,
			},
			expectState:      "sent_unverified",
			expectConfidence: 0.5,
		},
		{
			name: "Negative confidence (invalid)",
			focusEvidence: FocusVerificationEvidence{
				NodeStillExists: true,
				Confidence:      -0.5,
			},
			messageEvidence: SendVerificationEvidence{
				NewMessageNodes: 1,
				Confidence:      0.9,
			},
			expectState: "unknown",
			// -0.5*0.4 + 0.9*0.6 = -0.2 + 0.54 = 0.34
			expectConfidence: 0.34,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment := rules.AssessDeliveryState(tt.focusEvidence, tt.messageEvidence)

			if assessment.State != tt.expectState {
				t.Errorf("Expected state '%s', got '%s'", tt.expectState, assessment.State)
			}

			if assessment.Confidence != tt.expectConfidence {
				t.Errorf("Expected confidence %f, got %f", tt.expectConfidence, assessment.Confidence)
			}
		})
	}
}

func TestDeliveryAssessmentRules_AssessFocusOnlyState_DirtyData(t *testing.T) {
	rules := NewDeliveryAssessmentRules()

	tests := []struct {
		name          string
		focusEvidence FocusVerificationEvidence
		expectState   string
	}{
		{
			name: "Zero confidence",
			focusEvidence: FocusVerificationEvidence{
				Confidence: 0.0,
			},
			expectState: "unknown",
		},
		{
			name: "Negative confidence",
			focusEvidence: FocusVerificationEvidence{
				Confidence: -0.5,
			},
			expectState: "unknown",
		},
		{
			name: "Very high confidence",
			focusEvidence: FocusVerificationEvidence{
				Confidence: 1.5,
			},
			expectState: "verified",
		},
		{
			name: "Borderline verified (0.79)",
			focusEvidence: FocusVerificationEvidence{
				Confidence: 0.79,
			},
			expectState: "sent_unverified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment := rules.AssessFocusOnlyState(tt.focusEvidence)

			if assessment.State != tt.expectState {
				t.Errorf("Expected state '%s', got '%s'", tt.expectState, assessment.State)
			}
		})
	}
}

// ==================== DeliveryAssessmentRules Complex Scenarios ====================

func TestDeliveryAssessmentRules_ComplexScenarios(t *testing.T) {
	rules := NewDeliveryAssessmentRules()

	tests := []struct {
		name                string
		focusEvidence       FocusVerificationEvidence
		messageEvidence     SendVerificationEvidence
		expectedState       string
		expectedMinConf     float64
		expectedMaxConf     float64
	}{
		{
			name: "Perfect verification",
			focusEvidence: FocusVerificationEvidence{
				NodeStillExists:    true,
				NodeHasActiveState: true,
				TitleContainsTarget: true,
				PanelSwitchDetected: false,
				MessageAreaVisible: true,
				Confidence:         1.0,
				EvidenceCount:      4,
			},
			messageEvidence: SendVerificationEvidence{
				NewMessageNodes:   1,
				MessageNodeAdded:  true,
				MessageContentMatch: true,
				ScreenshotChanged: true,
				ChatAreaDiff:      0.05,
				Confidence:        1.0,
			},
			expectedState:   "verified",
			expectedMinConf: 0.8,
			expectedMaxConf: 1.0,
		},
		{
			name: "Partial verification",
			focusEvidence: FocusVerificationEvidence{
				NodeStillExists:    true,
				NodeHasActiveState: false,
				TitleContainsTarget: false,
				PanelSwitchDetected: false,
				MessageAreaVisible: false,
				Confidence:         0.4,
				EvidenceCount:      1,
			},
			messageEvidence: SendVerificationEvidence{
				NewMessageNodes:   1,
				MessageNodeAdded:  true,
				MessageContentMatch: false,
				ScreenshotChanged: false,
				ChatAreaDiff:      0.0,
				Confidence:        0.4,
			},
			expectedState:   "unknown",
			expectedMinConf: 0.0,
			expectedMaxConf: 0.5,
		},
		{
			name: "Sent but unverified",
			focusEvidence: FocusVerificationEvidence{
				NodeStillExists:    true,
				NodeHasActiveState: true,
				TitleContainsTarget: false,
				PanelSwitchDetected: false,
				MessageAreaVisible: true,
				Confidence:         0.6,
				EvidenceCount:      3,
			},
			messageEvidence: SendVerificationEvidence{
				NewMessageNodes:   1,
				MessageNodeAdded:  true,
				MessageContentMatch: true,
				ScreenshotChanged: true,
				ChatAreaDiff:      0.02,
				Confidence:        0.6,
			},
			expectedState:   "sent_unverified",
			expectedMinConf: 0.5,
			expectedMaxConf: 0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment := rules.AssessDeliveryState(tt.focusEvidence, tt.messageEvidence)

			if assessment.State != tt.expectedState {
				t.Errorf("Expected state '%s', got '%s'", tt.expectedState, assessment.State)
			}
			if assessment.Confidence < tt.expectedMinConf || assessment.Confidence > tt.expectedMaxConf {
				t.Errorf("Confidence %f not in expected range [%f, %f]",
					assessment.Confidence, tt.expectedMinConf, tt.expectedMaxConf)
			}
		})
	}
}
