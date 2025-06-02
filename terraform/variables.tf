variable "project_id" {
  description = "ID del proyecto de GCP"
  type        = string
}

variable "region" {
  description = "Región de GCP donde se desplegarán los servicios"
  type        = string
  default     = "us-central1"
}

variable "zone" {
  description = "Zona específica dentro de la región de GCP"
  type        = string
  default     = "us-central1-a"
}

variable "service_account_suffix" {
  description = "Sufijo para las cuentas de servicio"
  type        = string
  default     = "sa"
}

variable "bigquery_dataset" {
  description = "Nombre del dataset de BigQuery"
  type        = string
  default     = "kairosia_conversations"
}

variable "bigquery_table" {
  description = "Nombre de la tabla de BigQuery para almacenar las transcripciones"
  type        = string
  default     = "conversation_transcripts"
}

variable "firestore_collection" {
  description = "Nombre de la colección de Firestore para almacenar el estado de las conversaciones"
  type        = string
  default     = "conversation_states"
}

variable "vector_search_index_id" {
  description = "ID del índice de Vector Search"
  type        = string
  default     = "kairosia-conversation-index"
}

variable "vector_search_dimension" {
  description = "Dimensión del vector para Vector Search"
  type        = number
  default     = 768
}

variable "vector_search_distance_measure" {
  description = "Medida de distancia para Vector Search"
  type        = string
  default     = "COSINE"
}

variable "allow_unauthenticated" {
  description = "Permitir acceso no autenticado a los servicios de Cloud Run (solo para desarrollo)"
  type        = bool
  default     = true
}
