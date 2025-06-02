package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	speech "cloud.google.com/go/speech/apiv1"
	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/golang/protobuf/ptypes/struct"
	"github.com/twilio/twilio-go"
	dialogflow "google.golang.org/api/dialogflow/v3"
	"google.golang.org/api/option"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
	texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
	"google.golang.org/protobuf/types/known/structpb"

	"kairosia/internal/models"
	"kairosia/internal/utils"
)

var (
	projectID                  string
	region                     string
	dialogflowAgentID          string
	dialogflowLocation         string
	dialogflowDefaultLanguage  string
	sttLanguageCode            string
	sttModel                   string
	ttsLanguageCode            string
	ttsVoiceName               string
	ttsSpeakingRate            float64
	firestoreCollection        string
	vertexAIEmbeddingModel     string
	vertexAIVectorSearchIndex  string
	vertexAIVectorSearchDimension int
	vertexAIVectorSearchNeighbors int
	conversationHistoryServiceURL string
	transferPhoneNumber        string
)

func init() {
	// Inicializar variables de entorno
	projectID = utils.GetEnv("GCP_PROJECT_ID", "")
	region = utils.GetEnv("GCP_REGION", "us-central1")
	dialogflowAgentID = utils.GetEnv("DIALOGFLOW_AGENT_ID", "")
	dialogflowLocation = utils.GetEnv("DIALOGFLOW_LOCATION", "us-central1")
	dialogflowDefaultLanguage = utils.GetEnv("DIALOGFLOW_DEFAULT_LANGUAGE_CODE", "es-CL")
	sttLanguageCode = utils.GetEnv("STT_LANGUAGE_CODE", "es-CL")
	sttModel = utils.GetEnv("STT_MODEL", "phone_call")
	ttsLanguageCode = utils.GetEnv("TTS_LANGUAGE_CODE", "es-CL")
	ttsVoiceName = utils.GetEnv("TTS_VOICE_NAME", "es-CL-Standard-A")
	ttsSpeakingRate = utils.ParseFloat(utils.GetEnv("TTS_SPEAKING_RATE", "1.0"), 1.0)
	firestoreCollection = utils.GetEnv("FIRESTORE_COLLECTION", "conversation_states")
	vertexAIEmbeddingModel = utils.GetEnv("VERTEX_AI_EMBEDDING_MODEL", "textembedding-gecko")
	vertexAIVectorSearchIndex = utils.GetEnv("VERTEX_AI_VECTOR_SEARCH_INDEX", "")
	vertexAIVectorSearchDimension = utils.Atoi(utils.GetEnv("VERTEX_AI_VECTOR_SEARCH_DIMENSION", "768"), 768)
	vertexAIVectorSearchNeighbors = utils.Atoi(utils.GetEnv("VERTEX_AI_VECTOR_SEARCH_NEIGHBORS", "5"), 5)
	conversationHistoryServiceURL = utils.GetEnv("CONVERSATION_HISTORY_SERVICE_URL", "")
	transferPhoneNumber = utils.GetEnv("TRANSFER_PHONE_NUMBER", "+56912345678")

	// Registrar la función HTTP
	functions.HTTP("HandleVoiceRequest", HandleVoiceRequest)
}

// HandleVoiceRequest maneja las solicitudes de voz de Twilio
func HandleVoiceRequest(w http.ResponseWriter, r *http.Request) {
	// Parsear la solicitud de Twilio
	if err := r.ParseForm(); err != nil {
		log.Printf("Error al parsear el formulario: %v", err)
		http.Error(w, "Error al parsear el formulario", http.StatusBadRequest)
		return
	}

	// Crear una solicitud de voz a partir del formulario
	voiceRequest := &models.VoiceRequest{
		CallSid:    r.FormValue("CallSid"),
		AccountSid: r.FormValue("AccountSid"),
		From:       r.FormValue("From"),
		To:         r.FormValue("To"),
		Direction:  r.FormValue("Direction"),
		CallStatus: r.FormValue("CallStatus"),
		ApiVersion: r.FormValue("ApiVersion"),
		Digits:     r.FormValue("Digits"),
	}

	// Si hay un resultado de reconocimiento de voz, usarlo
	if speechResult := r.FormValue("SpeechResult"); speechResult != "" {
		voiceRequest.SpeechResult = speechResult
	}

	// Inicializar el contexto
	ctx := context.Background()

	// Obtener o crear el estado de la conversación
	conversationState, err := getOrCreateConversationState(ctx, voiceRequest)
	if err != nil {
		log.Printf("Error al obtener o crear el estado de la conversación: %v", err)
		respondWithError(w, err)
		return
	}

	// Si es una nueva llamada, responder con un saludo
	if conversationState.CurrentTurnIndex == 0 {
		// Generar un saludo inicial
		twiml := generateWelcomeTwiML()
		respondWithTwiML(w, twiml)
		return
	}

	// Procesar la entrada del usuario
	var userInput string
	if voiceRequest.SpeechResult != "" {
		// Si hay un resultado de reconocimiento de voz, usarlo
		userInput = voiceRequest.SpeechResult
	} else if voiceRequest.Digits != "" {
		// Si hay dígitos, usarlos
		userInput = fmt.Sprintf("Presionó %s", voiceRequest.Digits)
	} else {
		// Si no hay entrada, responder con un mensaje de error
		twiml := generateErrorTwiML("No se detectó ninguna entrada. Por favor, inténtelo de nuevo.")
		respondWithTwiML(w, twiml)
		return
	}

	// Crear una entrada de transcripción para el usuario
	userTranscriptEntry := models.TranscriptEntry{
		Speaker:    "user",
		Text:       userInput,
		Timestamp:  time.Now(),
		Confidence: 1.0, // Asumimos confianza máxima para simplificar
	}

	// Generar embedding para la entrada del usuario
	userEmbedding, err := generateEmbedding(ctx, userInput)
	if err != nil {
		log.Printf("Error al generar embedding para la entrada del usuario: %v", err)
		// Continuamos sin embedding
	} else {
		userTranscriptEntry.Embedding = userEmbedding
	}

	// Agregar la entrada del usuario a las entradas recientes
	conversationState.RecentTurns = append(conversationState.RecentTurns, userTranscriptEntry)
	conversationState.CurrentTurnIndex++

	// Buscar contexto relevante en Vector Search
	var contextText string
	if len(userEmbedding) > 0 {
		matches, err := searchVectorIndex(ctx, userEmbedding, conversationState.TenantID, conversationState.FromNumber)
		if err != nil {
			log.Printf("Error al buscar en el índice vectorial: %v", err)
			// Continuamos sin contexto adicional
		} else if len(matches) > 0 {
			// Construir el contexto a partir de los resultados de la búsqueda
			var contextBuilder strings.Builder
			contextBuilder.WriteString("Contexto adicional de conversaciones anteriores:\n")
			for _, match := range matches {
				contextBuilder.WriteString(fmt.Sprintf("- %s\n", match.Text))
			}
			contextText = contextBuilder.String()
			log.Printf("Contexto adicional encontrado: %s", contextText)
		}
	}

	// Consultar a Dialogflow CX
	dialogflowResponse, err := queryDialogflow(ctx, conversationState.DialogflowSessionID, userInput, contextText)
	if err != nil {
		log.Printf("Error al consultar a Dialogflow CX: %v", err)
		respondWithError(w, err)
		return
	}

	// Crear una entrada de transcripción para la IA
	aiTranscriptEntry := models.TranscriptEntry{
		Speaker:    "ai",
		Text:       dialogflowResponse.ResponseText,
		Timestamp:  time.Now(),
		Confidence: 1.0, // Asumimos confianza máxima para simplificar
	}

	// Generar embedding para la respuesta de la IA
	aiEmbedding, err := generateEmbedding(ctx, dialogflowResponse.ResponseText)
	if err != nil {
		log.Printf("Error al generar embedding para la respuesta de la IA: %v", err)
		// Continuamos sin embedding
	} else {
		aiTranscriptEntry.Embedding = aiEmbedding
	}

	// Agregar la entrada de la IA a las entradas recientes
	conversationState.RecentTurns = append(conversationState.RecentTurns, aiTranscriptEntry)
	conversationState.LastUpdateTimestamp = time.Now()

	// Verificar si hay un payload personalizado para transferir a un agente humano
	var handoffPayload *models.LiveAgentHandoffPayload
	if dialogflowResponse.CustomPayload != nil {
		if action, ok := dialogflowResponse.CustomPayload["action"].(string); ok && action == "LiveAgentHandoff" {
			// Parsear el payload de handoff
			handoffPayload = &models.LiveAgentHandoffPayload{
				Action:         action,
				TransferNumber: transferPhoneNumber, // Usar el número de transferencia configurado
				Reason:         "El cliente ha solicitado hablar con un agente humano",
				PreserveContext: true,
			}

			// Si hay un número de transferencia en el payload, usarlo
			if transferNumber, ok := dialogflowResponse.CustomPayload["transferNumber"].(string); ok && transferNumber != "" {
				handoffPayload.TransferNumber = transferNumber
			}

			// Si hay una razón en el payload, usarla
			if reason, ok := dialogflowResponse.CustomPayload["reason"].(string); ok && reason != "" {
				handoffPayload.Reason = reason
			}

			// Si hay un flag de preservar contexto en el payload, usarlo
			if preserveContext, ok := dialogflowResponse.CustomPayload["preserveContext"].(bool); ok {
				handoffPayload.PreserveContext = preserveContext
			}

			// Actualizar el estado de la conversación con la información de handoff
			now := time.Now()
			conversationState.HandoffOccurred = true
			conversationState.HandoffReason = handoffPayload.Reason
			conversationState.HandoffTimestamp = &now
		}
	}

	// Guardar el estado actualizado de la conversación
	if err := updateConversationState(ctx, conversationState); err != nil {
		log.Printf("Error al actualizar el estado de la conversación: %v", err)
		// Continuamos a pesar del error
	}

	// Enviar la transcripción completa al servicio de historial (en una goroutine para no bloquear)
	go func() {
		if err := sendTranscriptToHistoryService(conversationState, dialogflowResponse); err != nil {
			log.Printf("Error al enviar la transcripción al servicio de historial: %v", err)
		}
	}()

	// Generar la respuesta TwiML
	var twiml *models.TwiMLResponse
	if handoffPayload != nil {
		// Si hay un handoff, transferir la llamada
		twiml = generateHandoffTwiML(handoffPayload, dialogflowResponse.ResponseText)
	} else {
		// Si no hay handoff, generar una respuesta normal
		twiml = generateResponseTwiML(dialogflowResponse.ResponseText)
	}

	// Responder con TwiML
	respondWithTwiML(w, twiml)
}

// getOrCreateConversationState obtiene o crea el estado de una conversación
func getOrCreateConversationState(ctx context.Context, voiceRequest *models.VoiceRequest) (*models.ConversationState, error) {
	// Inicializar el cliente de Firestore
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("error al crear el cliente de Firestore: %v", err)
	}
	defer client.Close()

	// Obtener la referencia al documento
	docRef := client.Collection(firestoreCollection).Doc(voiceRequest.CallSid)

	// Intentar obtener el documento
	doc, err := docRef.Get(ctx)
	if err == nil {
		// Si el documento existe, convertirlo a ConversationState
		var state models.ConversationState
		if err := doc.DataTo(&state); err != nil {
			return nil, fmt.Errorf("error al convertir el documento a ConversationState: %v", err)
		}
		return &state, nil
	}

	// Si el documento no existe, crear uno nuevo
	now := time.Now()
	state := &models.ConversationState{
		CallSid:             voiceRequest.CallSid,
		TenantID:            "default", // Usar un tenant_id predeterminado
		FromNumber:          voiceRequest.From,
		ToNumber:            voiceRequest.To,
		StartTimestamp:      now,
		LastUpdateTimestamp: now,
		DialogflowSessionID: utils.GenerateSessionID(voiceRequest.CallSid),
		CurrentTurnIndex:    0,
		RecentTurns:         []models.TranscriptEntry{},
		HandoffOccurred:     false,
	}

	// Guardar el nuevo estado en Firestore
	if _, err := docRef.Set(ctx, state); err != nil {
		return nil, fmt.Errorf("error al guardar el estado de la conversación: %v", err)
	}

	return state, nil
}

// updateConversationState actualiza el estado de una conversación en Firestore
func updateConversationState(ctx context.Context, state *models.ConversationState) error {
	// Inicializar el cliente de Firestore
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("error al crear el cliente de Firestore: %v", err)
	}
	defer client.Close()

	// Obtener la referencia al documento
	docRef := client.Collection(firestoreCollection).Doc(state.CallSid)

	// Actualizar el documento
	_, err = docRef.Set(ctx, state)
	if err != nil {
		return fmt.Errorf("error al actualizar el estado de la conversación: %v", err)
	}

	return nil
}

// queryDialogflow consulta a Dialogflow CX
func queryDialogflow(ctx context.Context, sessionID, query, contextText string) (*models.DialogflowQueryResult, error) {
	// Inicializar el cliente de Dialogflow CX
	client, err := dialogflow.NewSessionsService(ctx, option.WithEndpoint(fmt.Sprintf("%s-dialogflow.googleapis.com:443", dialogflowLocation)))
	if err != nil {
		return nil, fmt.Errorf("error al crear el cliente de Dialogflow CX: %v", err)
	}

	// Construir la ruta de la sesión
	sessionPath := fmt.Sprintf("projects/%s/locations/%s/agents/%s/sessions/%s", projectID, dialogflowLocation, dialogflowAgentID, sessionID)

	// Construir la consulta
	queryInput := &dialogflow.QueryInput{
		Text: &dialogflow.TextInput{
			Text: query,
		},
		LanguageCode: dialogflowDefaultLanguage,
	}

	// Si hay contexto adicional, agregarlo como parámetros de la consulta
	var queryParams *dialogflow.QueryParameters
	if contextText != "" {
		contextStruct, err := structpb.NewStruct(map[string]interface{}{
			"additional_context": contextText,
		})
		if err != nil {
			log.Printf("Error al crear el struct de contexto adicional: %v", err)
		} else {
			queryParams = &dialogflow.QueryParameters{
				Parameters: contextStruct,
			}
		}
	}

	// Realizar la consulta
	request := &dialogflow.DetectIntentRequest{
		Session:     sessionPath,
		QueryInput:  queryInput,
		QueryParams: queryParams,
	}

	response, err := client.DetectIntent(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("error al detectar la intención: %v", err)
	}

	// Extraer la información relevante de la respuesta
	queryResult := &models.DialogflowQueryResult{
		SessionID:    sessionID,
		ResponseText: response.QueryResult.ResponseMessages[0].Text.Text[0],
	}

	// Extraer información adicional si está disponible
	if response.QueryResult.CurrentPage != nil {
		queryResult.PageID = response.QueryResult.CurrentPage.Name
	}

	if response.QueryResult.Match != nil {
		queryResult.IntentName = response.QueryResult.Match.Intent.DisplayName
		queryResult.IntentConfidence = response.QueryResult.Match.Confidence
	}

	if response.QueryResult.Parameters != nil {
		queryResult.Parameters = utils.ProtoStructToMap(response.QueryResult.Parameters.AsStruct())
	}

	// Extraer el payload personalizado si está disponible
	for _, message := range response.QueryResult.ResponseMessages {
		if message.Payload != nil {
			queryResult.CustomPayload = utils.ProtoStructToMap(message.Payload.AsStruct())
			break
		}
	}

	return queryResult, nil
}

// generateEmbedding genera un embedding para un texto
func generateEmbedding(ctx context.Context, text string) ([]float64, error) {
	// Nota: Esta es una implementación conceptual.
	// En un entorno de producción, se utilizaría la API de Vertex AI para generar embeddings.
	// Para simplificar, generamos un embedding aleatorio con la dimensión correcta.
	
	// En un entorno real, se utilizaría código como el siguiente:
	/*
	client, err := aiplatform.NewPredictionClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("error al crear el cliente de Vertex AI: %v", err)
	}
	defer client.Close()

	req := &aiplatformpb.PredictRequest{
		Endpoint: fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", projectID, region, vertexAIEmbeddingModel),
		Instances: []*structpb.Value{
			{
				Kind: &structpb.Value_StructValue{
					StructValue: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"content": {
								Kind: &structpb.Value_StringValue{
									StringValue: text,
								},
							},
						},
					},
				},
			},
		},
	}

	resp, err := client.Predict(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error al generar el embedding: %v", err)
	}

	// Extraer el embedding de la respuesta
	embedding := make([]float64, 0, vertexAIVectorSearchDimension)
	for _, value := range resp.Predictions[0].GetStructValue().Fields["embeddings"].GetListValue().Values {
		embedding = append(embedding, value.GetNumberValue())
	}
	*/

	// Para el MVP, generamos un embedding aleatorio
	embedding := make([]float64, vertexAIVectorSearchDimension)
	for i := range embedding {
		embedding[i] = 0.1 // Valor fijo para simplificar
	}

	return embedding, nil
}

// searchVectorIndex busca en el índice vectorial
func searchVectorIndex(ctx context.Context, embedding []float64, tenantID, fromNumber string) ([]models.VectorSearchMatch, error) {
	// Nota: Esta es una implementación conceptual.
	// En un entorno de producción, se utilizaría la API de Vertex AI Vector Search para buscar en el índice.
	// Para simplificar, devolvemos un resultado vacío.
	
	// En un entorno real, se utilizaría código como el siguiente:
	/*
	client, err := vectorsearch.NewIndexEndpointServiceClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("error al crear el cliente de Vector Search: %v", err)
	}
	defer client.Close()

	req := &vectorsearchpb.FindNeighborsRequest{
		IndexEndpoint: fmt.Sprintf("projects/%s/locations/%s/indexEndpoints/%s", projectID, region, vertexAIVectorSearchEndpoint),
		DeployedIndexId: vertexAIVectorSearchIndex,
		Queries: []*vectorsearchpb.Query{
			{
				Datapoint: &vectorsearchpb.Datapoint{
					FeatureVector: embedding,
				},
				NeighborCount: int32(vertexAIVectorSearchNeighbors),
				Parameters: &vectorsearchpb.Parameters{
					Filters: []*vectorsearchpb.Filter{
						{
							FilterType: &vectorsearchpb.Filter_StringFilter{
								StringFilter: &vectorsearchpb.StringFilter{
									Key: "tenant_id",
									Value: tenantID,
								},
							},
						},
						{
							FilterType: &vectorsearchpb.Filter_StringFilter{
								StringFilter: &vectorsearchpb.StringFilter{
									Key: "from_number",
									Value: fromNumber,
								},
							},
						},
					},
				},
			},
		},
	}

	resp, err := client.FindNeighbors(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error al buscar en el índice vectorial: %v", err)
	}

	// Extraer los resultados de la respuesta
	matches := make([]models.VectorSearchMatch, 0, len(resp.NearestNeighbors[0].Neighbors))
	for _, neighbor := range resp.NearestNeighbors[0].Neighbors {
		match := models.VectorSearchMatch{
			ID:       neighbor.DatapointId,
			Distance: neighbor.Distance,
			Text:     neighbor.Metadata["text"].GetStringValue(),
			Metadata: make(map[string]interface{}),
		}
		for k, v := range neighbor.Metadata {
			match.Metadata[k] = utils.ProtoValueToInterface(v)
		}
		matches = append(matches, match)
	}
	*/

	// Para el MVP, devolvemos un resultado vacío
	return []models.VectorSearchMatch{}, nil
}

// sendTranscriptToHistoryService envía la transcripción al servicio de historial
func sendTranscriptToHistoryService(state *models.ConversationState, dialogflowResult *models.DialogflowQueryResult) error {
	// Crear el payload
	now := time.Now()
	payload := &models.FullTranscriptPayload{
		CallSid:           state.CallSid,
		TenantID:          state.TenantID,
		FromNumber:        state.FromNumber,
		ToNumber:          state.ToNumber,
		StartTimestamp:    state.StartTimestamp,
		TranscriptEntries: state.RecentTurns,
		DialogflowMetadata: dialogflowResult,
		HandoffOccurred:   state.HandoffOccurred,
		HandoffReason:     state.HandoffReason,
		HandoffTimestamp:  state.HandoffTimestamp,
		CreatedAt:         now,
	}

	// Si la llamada ha terminado, calcular la duración
	if state.HandoffOccurred && state.HandoffTimestamp != nil {
		endTime := *state.HandoffTimestamp
		payload.EndTimestamp = &endTime
		payload.DurationSeconds = int(endTime.Sub(state.StartTimestamp).Seconds())
	}

	// Serializar el payload
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error al serializar el payload: %v", err)
	}

	// Enviar el payload al servicio de historial
	resp, err := http.Post(
		fmt.Sprintf("%s/save-transcript", conversationHistoryServiceURL),
		"application/json",
		strings.NewReader(string(jsonPayload)),
	)
	if err != nil {
		return fmt.Errorf("error al enviar el payload al servicio de historial: %v", err)
	}
	defer resp.Body.Close()

	// Verificar la respuesta
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error del servicio de historial: %s - %s", resp.Status, string(body))
	}

	return nil
}

// generateWelcomeTwiML genera el TwiML para el saludo inicial
func generateWelcomeTwiML() *models.TwiMLResponse {
	return &models.TwiMLResponse{
		Say: &models.TwiMLSay{
			Voice:    "Polly.Lupe",
			Language: ttsLanguageCode,
			Value:    "Hola, soy KairosIA, su asistente virtual. ¿En qué puedo ayudarle hoy?",
		},
		Gather: &models.TwiMLGather{
			Input:         "speech",
			Language:      sttLanguageCode,
			Timeout:       "5",
			SpeechTimeout: "auto",
			Say: &models.TwiMLSay{
				Voice:    "Polly.Lupe",
				Language: ttsLanguageCode,
				Value:    "Por favor, dígame en qué puedo ayudarle.",
			},
		},
	}
}

// generateResponseTwiML genera el TwiML para una respuesta normal
func generateResponseTwiML(responseText string) *models.TwiMLResponse {
	return &models.TwiMLResponse{
		Say: &models.TwiMLSay{
			Voice:    "Polly.Lupe",
			Language: ttsLanguageCode,
			Value:    responseText,
		},
		Gather: &models.TwiMLGather{
			Input:         "speech",
			Language:      sttLanguageCode,
			Timeout:       "5",
			SpeechTimeout: "auto",
		},
	}
}

// generateHandoffTwiML genera el TwiML para transferir a un agente humano
func generateHandoffTwiML(handoffPayload *models.LiveAgentHandoffPayload, responseText string) *models.TwiMLResponse {
	return &models.TwiMLResponse{
		Say: &models.TwiMLSay{
			Voice:    "Polly.Lupe",
			Language: ttsLanguageCode,
			Value:    responseText + " Le transferiré con un agente humano. Por favor, espere un momento.",
		},
		Dial: &models.TwiMLDial{
			CallerId: "{{From}}",
			Number:   handoffPayload.TransferNumber,
		},
	}
}

// generateErrorTwiML genera el TwiML para un mensaje de error
func generateErrorTwiML(errorMessage string) *models.TwiMLResponse {
	return &models.TwiMLResponse{
		Say: &models.TwiMLSay{
			Voice:    "Polly.Lupe",
			Language: ttsLanguageCode,
			Value:    errorMessage,
		},
		Gather: &models.TwiMLGather{
			Input:         "speech",
			Language:      sttLanguageCode,
			Timeout:       "5",
			SpeechTimeout: "auto",
		},
	}
}

// respondWithTwiML responde con TwiML
func respondWithTwiML(w http.ResponseWriter, twiml *models.TwiMLResponse) {
	// Serializar el TwiML
	xmlData, err := xml.MarshalIndent(twiml, "", "  ")
	if err != nil {
		log.Printf("Error al serializar el TwiML: %v", err)
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}

	// Agregar la declaración XML
	xmlString := xml.Header + string(xmlData)

	// Establecer las cabeceras
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Length", strconv.Itoa(len(xmlString)))

	// Escribir la respuesta
	if _, err := w.Write([]byte(xmlString)); err != nil {
		log.Printf("Error al escribir la respuesta: %v", err)
	}
}

// respondWithError responde con un mensaje de error
func respondWithError(w http.ResponseWriter, err error) {
	log.Printf("Error: %v", err)
	twiml := generateErrorTwiML("Lo siento, ha ocurrido un error. Por favor, inténtelo de nuevo más tarde.")
	respondWithTwiML(w, twiml)
}

func main() {
	// Obtener el puerto del entorno o usar 8080 por defecto
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Iniciar el servidor HTTP
	log.Printf("Iniciando servidor en el puerto %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
