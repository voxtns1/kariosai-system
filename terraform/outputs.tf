output "voice_orchestration_service_url" {
  description = "URL del servicio de orquestación de voz"
  value       = google_cloud_run_service.voice_orchestration_service.status[0].url
}

output "conversation_history_service_url" {
  description = "URL del servicio de historial de conversaciones"
  value       = google_cloud_run_service.conversation_history_service.status[0].url
}

output "voice_orchestration_service_account" {
  description = "Cuenta de servicio para el servicio de orquestación de voz"
  value       = google_service_account.voice_orchestration_sa.email
}

output "history_service_service_account" {
  description = "Cuenta de servicio para el servicio de historial de conversaciones"
  value       = google_service_account.history_service_sa.email
}

output "bigquery_dataset_id" {
  description = "ID del dataset de BigQuery"
  value       = google_bigquery_dataset.conversations_dataset.dataset_id
}

output "bigquery_table_id" {
  description = "ID de la tabla de BigQuery"
  value       = google_bigquery_table.conversation_transcripts.table_id
}

output "firestore_database" {
  description = "Base de datos de Firestore"
  value       = google_firestore_database.database.name
}
