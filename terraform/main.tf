provider "google" {
  project = var.project_id
  region  = var.region
  zone    = var.zone
}

# Habilitar las APIs necesarias
resource "google_project_service" "required_apis" {
  for_each = toset([
    "compute.googleapis.com",
    "containerregistry.googleapis.com",
    "run.googleapis.com",
    "dialogflow.googleapis.com",
    "speech.googleapis.com",
    "texttospeech.googleapis.com",
    "firestore.googleapis.com",
    "bigquery.googleapis.com",
    "aiplatform.googleapis.com",
    "iam.googleapis.com"
  ])
  
  project = var.project_id
  service = each.key
  
  disable_dependent_services = false
  disable_on_destroy         = false
}

# Crear cuentas de servicio
resource "google_service_account" "voice_orchestration_sa" {
  account_id   = "voice-orchestration-${var.service_account_suffix}"
  display_name = "Voice Orchestration Service Account"
  description  = "Cuenta de servicio para el servicio de orquestación de voz"
  
  depends_on = [google_project_service.required_apis]
}

resource "google_service_account" "history_service_sa" {
  account_id   = "history-service-${var.service_account_suffix}"
  display_name = "Conversation History Service Account"
  description  = "Cuenta de servicio para el servicio de historial de conversaciones"
  
  depends_on = [google_project_service.required_apis]
}

# Asignar roles IAM a las cuentas de servicio
resource "google_project_iam_member" "voice_orchestration_roles" {
  for_each = toset([
    "roles/dialogflow.client",
    "roles/speech.client",
    "roles/texttospeech.client",
    "roles/firestore.user",
    "roles/aiplatform.user"
  ])
  
  project = var.project_id
  role    = each.key
  member  = "serviceAccount:${google_service_account.voice_orchestration_sa.email}"
}

resource "google_project_iam_member" "history_service_roles" {
  for_each = toset([
    "roles/bigquery.dataEditor",
    "roles/aiplatform.user"
  ])
  
  project = var.project_id
  role    = each.key
  member  = "serviceAccount:${google_service_account.history_service_sa.email}"
}

# Permitir que el servicio de orquestación invoque al servicio de historial
resource "google_service_account_iam_member" "voice_orchestration_invoker" {
  service_account_id = google_service_account.history_service_sa.name
  role               = "roles/run.invoker"
  member             = "serviceAccount:${google_service_account.voice_orchestration_sa.email}"
}

# Crear dataset de BigQuery
resource "google_bigquery_dataset" "conversations_dataset" {
  dataset_id                  = var.bigquery_dataset
  friendly_name               = "KairosIA Conversations"
  description                 = "Dataset para almacenar las transcripciones de las conversaciones de KairosIA"
  location                    = var.region
  default_table_expiration_ms = null
  
  depends_on = [google_project_service.required_apis]
}

# Crear tabla de BigQuery
resource "google_bigquery_table" "conversation_transcripts" {
  dataset_id = google_bigquery_dataset.conversations_dataset.dataset_id
  table_id   = var.bigquery_table
  
  schema = <<EOF
[
  {
    "name": "call_sid",
    "type": "STRING",
    "mode": "REQUIRED",
    "description": "ID único de la llamada en Twilio"
  },
  {
    "name": "tenant_id",
    "type": "STRING",
    "mode": "REQUIRED",
    "description": "ID del inquilino (empresa)"
  },
  {
    "name": "from_number",
    "type": "STRING",
    "mode": "REQUIRED",
    "description": "Número de teléfono del llamante"
  },
  {
    "name": "to_number",
    "type": "STRING",
    "mode": "REQUIRED",
    "description": "Número de teléfono del destinatario"
  },
  {
    "name": "start_timestamp",
    "type": "TIMESTAMP",
    "mode": "REQUIRED",
    "description": "Marca de tiempo de inicio de la llamada"
  },
  {
    "name": "end_timestamp",
    "type": "TIMESTAMP",
    "mode": "NULLABLE",
    "description": "Marca de tiempo de finalización de la llamada"
  },
  {
    "name": "duration_seconds",
    "type": "INTEGER",
    "mode": "NULLABLE",
    "description": "Duración de la llamada en segundos"
  },
  {
    "name": "transcript_entries",
    "type": "RECORD",
    "mode": "REPEATED",
    "description": "Entradas de la transcripción",
    "fields": [
      {
        "name": "speaker",
        "type": "STRING",
        "mode": "REQUIRED",
        "description": "Identificador del hablante (user o ai)"
      },
      {
        "name": "text",
        "type": "STRING",
        "mode": "REQUIRED",
        "description": "Texto transcrito"
      },
      {
        "name": "timestamp",
        "type": "TIMESTAMP",
        "mode": "REQUIRED",
        "description": "Marca de tiempo de la entrada"
      },
      {
        "name": "confidence",
        "type": "FLOAT",
        "mode": "NULLABLE",
        "description": "Confianza de la transcripción"
      },
      {
        "name": "embedding",
        "type": "FLOAT",
        "mode": "REPEATED",
        "description": "Vector de embedding de la entrada"
      }
    ]
  },
  {
    "name": "dialogflow_metadata",
    "type": "RECORD",
    "mode": "NULLABLE",
    "description": "Metadatos de Dialogflow CX",
    "fields": [
      {
        "name": "session_id",
        "type": "STRING",
        "mode": "REQUIRED",
        "description": "ID de la sesión de Dialogflow CX"
      },
      {
        "name": "flow_id",
        "type": "STRING",
        "mode": "NULLABLE",
        "description": "ID del flujo de Dialogflow CX"
      },
      {
        "name": "intent_name",
        "type": "STRING",
        "mode": "NULLABLE",
        "description": "Nombre de la intención detectada"
      },
      {
        "name": "intent_confidence",
        "type": "FLOAT",
        "mode": "NULLABLE",
        "description": "Confianza de la intención detectada"
      },
      {
        "name": "parameters",
        "type": "STRING",
        "mode": "NULLABLE",
        "description": "Parámetros extraídos en formato JSON"
      },
      {
        "name": "page_id",
        "type": "STRING",
        "mode": "NULLABLE",
        "description": "ID de la página de Dialogflow CX"
      }
    ]
  },
  {
    "name": "handoff_occurred",
    "type": "BOOLEAN",
    "mode": "REQUIRED",
    "description": "Indica si se produjo una transferencia a un agente humano"
  },
  {
    "name": "handoff_reason",
    "type": "STRING",
    "mode": "NULLABLE",
    "description": "Razón de la transferencia a un agente humano"
  },
  {
    "name": "handoff_timestamp",
    "type": "TIMESTAMP",
    "mode": "NULLABLE",
    "description": "Marca de tiempo de la transferencia a un agente humano"
  },
  {
    "name": "embedding",
    "type": "FLOAT",
    "mode": "REPEATED",
    "description": "Vector de embedding de la conversación completa"
  },
  {
    "name": "created_at",
    "type": "TIMESTAMP",
    "mode": "REQUIRED",
    "description": "Marca de tiempo de creación del registro"
  }
]
EOF
  
  depends_on = [google_bigquery_dataset.conversations_dataset]
}

# Crear base de datos de Firestore
resource "google_firestore_database" "database" {
  name        = "(default)"
  location_id = var.region
  type        = "FIRESTORE_NATIVE"
  
  depends_on = [google_project_service.required_apis]
}

# Desplegar servicio de orquestación de voz en Cloud Run
resource "google_cloud_run_service" "voice_orchestration_service" {
  name     = "voice-orchestration-service"
  location = var.region
  
  template {
    spec {
      containers {
        image = "gcr.io/${var.project_id}/voice-orchestration-service:latest"
        
        env {
          name  = "GCP_PROJECT_ID"
          value = var.project_id
        }
        
        env {
          name  = "GCP_REGION"
          value = var.region
        }
        
        env {
          name  = "DIALOGFLOW_AGENT_ID"
          value = "your-dialogflow-agent-id" # Reemplazar con variable de entorno
        }
        
        env {
          name  = "DIALOGFLOW_LOCATION"
          value = var.region
        }
        
        env {
          name  = "DIALOGFLOW_DEFAULT_LANGUAGE_CODE"
          value = "es-CL"
        }
        
        env {
          name  = "STT_LANGUAGE_CODE"
          value = "es-CL"
        }
        
        env {
          name  = "STT_MODEL"
          value = "phone_call"
        }
        
        env {
          name  = "TTS_LANGUAGE_CODE"
          value = "es-CL"
        }
        
        env {
          name  = "TTS_VOICE_NAME"
          value = "es-CL-Standard-A"
        }
        
        env {
          name  = "TTS_SPEAKING_RATE"
          value = "1.0"
        }
        
        env {
          name  = "FIRESTORE_COLLECTION"
          value = var.firestore_collection
        }
        
        env {
          name  = "VERTEX_AI_EMBEDDING_MODEL"
          value = "textembedding-gecko"
        }
        
        env {
          name  = "VERTEX_AI_VECTOR_SEARCH_INDEX"
          value = var.vector_search_index_id
        }
        
        env {
          name  = "VERTEX_AI_VECTOR_SEARCH_DIMENSION"
          value = var.vector_search_dimension
        }
        
        env {
          name  = "VERTEX_AI_VECTOR_SEARCH_NEIGHBORS"
          value = "5"
        }
        
        env {
          name  = "CONVERSATION_HISTORY_SERVICE_URL"
          value = google_cloud_run_service.conversation_history_service.status[0].url
        }
        
        env {
          name  = "TRANSFER_PHONE_NUMBER"
          value = "+56912345678" # Reemplazar con variable de entorno
        }
      }
      
      service_account_name = google_service_account.voice_orchestration_sa.email
    }
  }
  
  traffic {
    percent         = 100
    latest_revision = true
  }
  
  depends_on = [
    google_project_service.required_apis,
    google_service_account.voice_orchestration_sa,
    google_cloud_run_service.conversation_history_service
  ]
}

# Desplegar servicio de historial de conversaciones en Cloud Run
resource "google_cloud_run_service" "conversation_history_service" {
  name     = "conversation-history-service"
  location = var.region
  
  template {
    spec {
      containers {
        image = "gcr.io/${var.project_id}/conversation-history-service:latest"
        
        env {
          name  = "GCP_PROJECT_ID"
          value = var.project_id
        }
        
        env {
          name  = "GCP_REGION"
          value = var.region
        }
        
        env {
          name  = "BIGQUERY_DATASET"
          value = var.bigquery_dataset
        }
        
        env {
          name  = "BIGQUERY_TABLE"
          value = var.bigquery_table
        }
        
        env {
          name  = "VERTEX_AI_EMBEDDING_MODEL"
          value = "textembedding-gecko"
        }
        
        env {
          name  = "VERTEX_AI_VECTOR_SEARCH_INDEX"
          value = var.vector_search_index_id
        }
      }
      
      service_account_name = google_service_account.history_service_sa.email
    }
  }
  
  traffic {
    percent         = 100
    latest_revision = true
  }
  
  depends_on = [
    google_project_service.required_apis,
    google_service_account.history_service_sa,
    google_bigquery_table.conversation_transcripts
  ]
}

# Configurar permisos de acceso para los servicios de Cloud Run
resource "google_cloud_run_service_iam_member" "voice_orchestration_public" {
  count    = var.allow_unauthenticated ? 1 : 0
  location = google_cloud_run_service.voice_orchestration_service.location
  project  = google_cloud_run_service.voice_orchestration_service.project
  service  = google_cloud_run_service.voice_orchestration_service.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_cloud_run_service_iam_member" "conversation_history_public" {
  count    = var.allow_unauthenticated ? 1 : 0
  location = google_cloud_run_service.conversation_history_service.location
  project  = google_cloud_run_service.conversation_history_service.project
  service  = google_cloud_run_service.conversation_history_service.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# Nota conceptual sobre Vertex AI Vector Search
# En un entorno de producción, se recomienda crear el índice de Vector Search
# y el endpoint a través de la API de Vertex AI o la consola de GCP.
# Para un MVP, esto puede hacerse manualmente o mediante scripts separados.
# El siguiente bloque es solo una referencia conceptual.

/*
resource "google_vertex_ai_index" "conversation_index" {
  display_name = var.vector_search_index_id
  description  = "Índice vectorial para conversaciones de KairosIA"
  region       = var.region
  
  metadata {
    contents_delta_uri = "gs://${var.project_id}-vertex-ai/indexes/${var.vector_search_index_id}/contents"
    config {
      dimensions = var.vector_search_dimension
      approximate_neighbors_count = 150
      distance_measure_type = var.vector_search_distance_measure
      algorithm_config {
        tree_ah_config {
          leaf_node_embedding_count = 1000
          leaf_nodes_to_search_percent = 10
        }
      }
    }
  }
  
  depends_on = [google_project_service.required_apis]
}

resource "google_vertex_ai_index_endpoint" "conversation_endpoint" {
  display_name = "${var.vector_search_index_id}-endpoint"
  description  = "Endpoint para el índice vectorial de conversaciones de KairosIA"
  region       = var.region
  
  depends_on = [google_vertex_ai_index.conversation_index]
}
*/
