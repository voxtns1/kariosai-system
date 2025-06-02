package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"google.golang.org/api/option"

	"kairosia/internal/models"
	"kairosia/internal/utils"
)

var (
	projectID                 string
	region                    string
	bigqueryDataset           string
	bigqueryTable             string
	vertexAIEmbeddingModel    string
	vertexAIVectorSearchIndex string
)

func init() {
	// Inicializar variables de entorno
	projectID = utils.GetEnv("GCP_PROJECT_ID", "")
	region = utils.GetEnv("GCP_REGION", "us-central1")
	bigqueryDataset = utils.GetEnv("BIGQUERY_DATASET", "kairosia_conversations")
	bigqueryTable = utils.GetEnv("BIGQUERY_TABLE", "conversation_transcripts")
	vertexAIEmbeddingModel = utils.GetEnv("VERTEX_AI_EMBEDDING_MODEL", "textembedding-gecko")
	vertexAIVectorSearchIndex = utils.GetEnv("VERTEX_AI_VECTOR_SEARCH_INDEX", "")

	// Registrar la función HTTP
	functions.HTTP("SaveTranscript", SaveTranscript)
}

// SaveTranscript guarda la transcripción de una conversación en BigQuery
func SaveTranscript(w http.ResponseWriter, r *http.Request) {
	// Verificar que el método sea POST
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	// Leer el cuerpo de la solicitud
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error al leer el cuerpo de la solicitud: %v", err)
		http.Error(w, "Error al leer el cuerpo de la solicitud", http.StatusBadRequest)
		return
	}

	// Parsear el payload
	var payload models.FullTranscriptPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("Error al parsear el payload: %v", err)
		http.Error(w, "Error al parsear el payload", http.StatusBadRequest)
		return
	}

	// Inicializar el contexto
	ctx := context.Background()

	// Guardar la transcripción en BigQuery
	if err := saveTranscriptToBigQuery(ctx, &payload); err != nil {
		log.Printf("Error al guardar la transcripción en BigQuery: %v", err)
		http.Error(w, fmt.Sprintf("Error al guardar la transcripción en BigQuery: %v", err), http.StatusInternalServerError)
		return
	}

	// Actualizar el índice vectorial (conceptual)
	if err := updateVectorIndex(ctx, &payload); err != nil {
		log.Printf("Error al actualizar el índice vectorial: %v", err)
		// No fallamos la solicitud por un error en la actualización del índice
	}

	// Responder con éxito
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// saveTranscriptToBigQuery guarda la transcripción en BigQuery
func saveTranscriptToBigQuery(ctx context.Context, payload *models.FullTranscriptPayload) error {
	// Inicializar el cliente de BigQuery
	client, err := bigquery.NewClient(ctx, projectID, option.WithEndpoint(fmt.Sprintf("https://bigquery.googleapis.com")))
	if err != nil {
		return fmt.Errorf("error al crear el cliente de BigQuery: %v", err)
	}
	defer client.Close()

	// Obtener la referencia a la tabla
	table := client.Dataset(bigqueryDataset).Table(bigqueryTable)

	// Insertar el registro
	inserter := table.Inserter()
	if err := inserter.Put(ctx, payload); err != nil {
		return fmt.Errorf("error al insertar el registro en BigQuery: %v", err)
	}

	return nil
}

// updateVectorIndex actualiza el índice vectorial con los embeddings de la conversación
func updateVectorIndex(ctx context.Context, payload *models.FullTranscriptPayload) error {
	// Nota: Esta es una implementación conceptual.
	// En un entorno de producción, se utilizaría la API de Vertex AI Vector Search para actualizar el índice.
	// Para simplificar, no hacemos nada en el MVP.
	
	// En un entorno real, se utilizaría código como el siguiente:
	/*
	client, err := vectorsearch.NewIndexEndpointServiceClient(ctx)
	if err != nil {
		return fmt.Errorf("error al crear el cliente de Vector Search: %v", err)
	}
	defer client.Close()

	// Preparar los datapoints para cada entrada de la transcripción
	datapoints := make([]*vectorsearchpb.Datapoint, 0, len(payload.TranscriptEntries))
	for i, entry := range payload.TranscriptEntries {
		if len(entry.Embedding) == 0 {
			continue
		}

		datapoint := &vectorsearchpb.Datapoint{
			DatapointId: fmt.Sprintf("%s-%d", payload.CallSid, i),
			FeatureVector: entry.Embedding,
			Metadata: map[string]*structpb.Value{
				"call_sid": {
					Kind: &structpb.Value_StringValue{
						StringValue: payload.CallSid,
					},
				},
				"tenant_id": {
					Kind: &structpb.Value_StringValue{
						StringValue: payload.TenantID,
					},
				},
				"from_number": {
					Kind: &structpb.Value_StringValue{
						StringValue: payload.FromNumber,
					},
				},
				"speaker": {
					Kind: &structpb.Value_StringValue{
						StringValue: entry.Speaker,
					},
				},
				"text": {
					Kind: &structpb.Value_StringValue{
						StringValue: entry.Text,
					},
				},
				"timestamp": {
					Kind: &structpb.Value_StringValue{
						StringValue: entry.Timestamp.Format(time.RFC3339),
					},
				},
			},
		}
		datapoints = append(datapoints, datapoint)
	}

	// Si hay un embedding para toda la conversación, agregarlo también
	if len(payload.Embedding) > 0 {
		datapoint := &vectorsearchpb.Datapoint{
			DatapointId: fmt.Sprintf("%s-full", payload.CallSid),
			FeatureVector: payload.Embedding,
			Metadata: map[string]*structpb.Value{
				"call_sid": {
					Kind: &structpb.Value_StringValue{
						StringValue: payload.CallSid,
					},
				},
				"tenant_id": {
					Kind: &structpb.Value_StringValue{
						StringValue: payload.TenantID,
					},
				},
				"from_number": {
					Kind: &structpb.Value_StringValue{
						StringValue: payload.FromNumber,
					},
				},
				"is_full_conversation": {
					Kind: &structpb.Value_BoolValue{
						BoolValue: true,
					},
				},
				"text": {
					Kind: &structpb.Value_StringValue{
						StringValue: "Conversación completa: " + strings.Join(getAllTexts(payload.TranscriptEntries), " "),
					},
				},
				"timestamp": {
					Kind: &structpb.Value_StringValue{
						StringValue: payload.StartTimestamp.Format(time.RFC3339),
					},
				},
			},
		}
		datapoints = append(datapoints, datapoint)
	}

	// Actualizar el índice
	req := &vectorsearchpb.UpsertDatapointsRequest{
		IndexEndpoint: fmt.Sprintf("projects/%s/locations/%s/indexEndpoints/%s", projectID, region, vertexAIVectorSearchEndpoint),
		DeployedIndexId: vertexAIVectorSearchIndex,
		Datapoints: datapoints,
	}

	_, err = client.UpsertDatapoints(ctx, req)
	if err != nil {
		return fmt.Errorf("error al actualizar el índice vectorial: %v", err)
	}
	*/

	return nil
}

// getAllTexts obtiene todos los textos de las entradas de la transcripción
func getAllTexts(entries []models.TranscriptEntry) []string {
	texts := make([]string, len(entries))
	for i, entry := range entries {
		texts[i] = entry.Text
	}
	return texts
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
