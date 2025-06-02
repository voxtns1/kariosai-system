package models

import (
	"time"
)

// VoiceRequest representa la solicitud de voz de Twilio
type VoiceRequest struct {
	CallSid       string `json:"CallSid"`
	AccountSid    string `json:"AccountSid"`
	From          string `json:"From"`
	To            string `json:"To"`
	Direction     string `json:"Direction"`
	CallStatus    string `json:"CallStatus"`
	ApiVersion    string `json:"ApiVersion"`
	RecordingUrl  string `json:"RecordingUrl,omitempty"`
	RecordingSid  string `json:"RecordingSid,omitempty"`
	RecordingDuration string `json:"RecordingDuration,omitempty"`
	Digits        string `json:"Digits,omitempty"`
	SpeechResult  string `json:"SpeechResult,omitempty"`
}

// ConversationState representa el estado de una conversación en Firestore
type ConversationState struct {
	CallSid        string    `json:"call_sid" firestore:"call_sid"`
	TenantID       string    `json:"tenant_id" firestore:"tenant_id"`
	FromNumber     string    `json:"from_number" firestore:"from_number"`
	ToNumber       string    `json:"to_number" firestore:"to_number"`
	StartTimestamp time.Time `json:"start_timestamp" firestore:"start_timestamp"`
	LastUpdateTimestamp time.Time `json:"last_update_timestamp" firestore:"last_update_timestamp"`
	DialogflowSessionID string `json:"dialogflow_session_id" firestore:"dialogflow_session_id"`
	CurrentTurnIndex int    `json:"current_turn_index" firestore:"current_turn_index"`
	RecentTurns     []TranscriptEntry `json:"recent_turns" firestore:"recent_turns"`
	HandoffOccurred bool   `json:"handoff_occurred" firestore:"handoff_occurred"`
	HandoffReason   string `json:"handoff_reason,omitempty" firestore:"handoff_reason,omitempty"`
	HandoffTimestamp *time.Time `json:"handoff_timestamp,omitempty" firestore:"handoff_timestamp,omitempty"`
}

// TranscriptEntry representa una entrada en la transcripción de una conversación
type TranscriptEntry struct {
	Speaker    string    `json:"speaker" firestore:"speaker" bigquery:"speaker"`
	Text       string    `json:"text" firestore:"text" bigquery:"text"`
	Timestamp  time.Time `json:"timestamp" firestore:"timestamp" bigquery:"timestamp"`
	Confidence float64   `json:"confidence,omitempty" firestore:"confidence,omitempty" bigquery:"confidence"`
	Embedding  []float64 `json:"embedding,omitempty" firestore:"embedding,omitempty" bigquery:"embedding"`
}

// DialogflowQueryResult representa el resultado de una consulta a Dialogflow CX
type DialogflowQueryResult struct {
	SessionID        string                 `json:"session_id" firestore:"session_id" bigquery:"session_id"`
	FlowID           string                 `json:"flow_id,omitempty" firestore:"flow_id,omitempty" bigquery:"flow_id"`
	IntentName       string                 `json:"intent_name,omitempty" firestore:"intent_name,omitempty" bigquery:"intent_name"`
	IntentConfidence float64                `json:"intent_confidence,omitempty" firestore:"intent_confidence,omitempty" bigquery:"intent_confidence"`
	Parameters       map[string]interface{} `json:"parameters,omitempty" firestore:"parameters,omitempty" bigquery:"parameters"`
	PageID           string                 `json:"page_id,omitempty" firestore:"page_id,omitempty" bigquery:"page_id"`
	ResponseText     string                 `json:"response_text" firestore:"response_text"`
	CustomPayload    map[string]interface{} `json:"custom_payload,omitempty" firestore:"custom_payload,omitempty"`
}

// LiveAgentHandoffPayload representa el payload para la transferencia a un agente humano
type LiveAgentHandoffPayload struct {
	Action         string `json:"action"`
	TransferNumber string `json:"transferNumber"`
	Reason         string `json:"reason"`
	PreserveContext bool   `json:"preserveContext"`
}

// FullTranscriptPayload representa el payload completo para guardar en BigQuery
type FullTranscriptPayload struct {
	CallSid           string             `json:"call_sid" bigquery:"call_sid"`
	TenantID          string             `json:"tenant_id" bigquery:"tenant_id"`
	FromNumber        string             `json:"from_number" bigquery:"from_number"`
	ToNumber          string             `json:"to_number" bigquery:"to_number"`
	StartTimestamp    time.Time          `json:"start_timestamp" bigquery:"start_timestamp"`
	EndTimestamp      *time.Time         `json:"end_timestamp,omitempty" bigquery:"end_timestamp"`
	DurationSeconds   int                `json:"duration_seconds,omitempty" bigquery:"duration_seconds"`
	TranscriptEntries []TranscriptEntry  `json:"transcript_entries" bigquery:"transcript_entries"`
	DialogflowMetadata *DialogflowQueryResult `json:"dialogflow_metadata,omitempty" bigquery:"dialogflow_metadata"`
	HandoffOccurred   bool               `json:"handoff_occurred" bigquery:"handoff_occurred"`
	HandoffReason     string             `json:"handoff_reason,omitempty" bigquery:"handoff_reason"`
	HandoffTimestamp  *time.Time         `json:"handoff_timestamp,omitempty" bigquery:"handoff_timestamp"`
	Embedding         []float64          `json:"embedding,omitempty" bigquery:"embedding"`
	CreatedAt         time.Time          `json:"created_at" bigquery:"created_at"`
}

// VectorSearchMatch representa un resultado de búsqueda de Vector Search
type VectorSearchMatch struct {
	ID        string                 `json:"id"`
	Distance  float64                `json:"distance"`
	Text      string                 `json:"text"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// VectorSearchRequest representa una solicitud a Vector Search
type VectorSearchRequest struct {
	Embedding []float64               `json:"embedding"`
	Limit     int                     `json:"limit"`
	Filters   map[string]interface{}  `json:"filters,omitempty"`
}

// VectorSearchResponse representa una respuesta de Vector Search
type VectorSearchResponse struct {
	Matches []VectorSearchMatch `json:"matches"`
}

// TwiMLResponse representa una respuesta TwiML para Twilio
type TwiMLResponse struct {
	XMLName struct{} `xml:"Response"`
	Say     *TwiMLSay `xml:"Say,omitempty"`
	Gather  *TwiMLGather `xml:"Gather,omitempty"`
	Dial    *TwiMLDial `xml:"Dial,omitempty"`
	Hangup  *TwiMLHangup `xml:"Hangup,omitempty"`
}

// TwiMLSay representa el elemento Say de TwiML
type TwiMLSay struct {
	Voice    string `xml:"voice,attr,omitempty"`
	Language string `xml:"language,attr,omitempty"`
	Value    string `xml:",chardata"`
}

// TwiMLGather representa el elemento Gather de TwiML
type TwiMLGather struct {
	Input         string `xml:"input,attr"`
	Timeout       string `xml:"timeout,attr,omitempty"`
	NumDigits     string `xml:"numDigits,attr,omitempty"`
	Action        string `xml:"action,attr,omitempty"`
	Method        string `xml:"method,attr,omitempty"`
	Language      string `xml:"language,attr,omitempty"`
	Hints         string `xml:"hints,attr,omitempty"`
	ProfanityFilter string `xml:"profanityFilter,attr,omitempty"`
	SpeechTimeout string `xml:"speechTimeout,attr,omitempty"`
	Say           *TwiMLSay `xml:"Say,omitempty"`
}

// TwiMLDial representa el elemento Dial de TwiML
type TwiMLDial struct {
	Action      string `xml:"action,attr,omitempty"`
	Method      string `xml:"method,attr,omitempty"`
	Timeout     string `xml:"timeout,attr,omitempty"`
	CallerId    string `xml:"callerId,attr,omitempty"`
	Record      string `xml:"record,attr,omitempty"`
	Number      string `xml:",chardata"`
}

// TwiMLHangup representa el elemento Hangup de TwiML
type TwiMLHangup struct {
}
