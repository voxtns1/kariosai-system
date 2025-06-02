# KairosIA - Sistema de Atención Telefónica Conversacional basado en IA

KairosIA es un sistema de atención telefónica conversacional basado en IA diseñado para transformar la interacción con clientes, comenzando por empresas de telecomunicaciones en Chile. Este sistema ofrece una atención natural, contextual y resolutiva mediante la integración de tecnologías de voz y procesamiento de lenguaje natural.

Este documento proporciona una guía completa para montar y desplegar el proyecto KairosIA en Google Cloud Platform (GCP).

## Sección 1: Requisitos Previos

Antes de comenzar con la instalación y configuración de KairosIA, asegúrate de tener instalados y configurados los siguientes componentes:

- **Google Cloud SDK (gcloud CLI)**: Herramienta de línea de comandos para interactuar con los servicios de Google Cloud.
  - [Instrucciones de instalación](https://cloud.google.com/sdk/docs/install)
  - Ejecuta `gcloud auth login` para autenticarte con tu cuenta de Google Cloud.

- **Docker**: Plataforma para desarrollar, enviar y ejecutar aplicaciones en contenedores.
  - [Instrucciones de instalación](https://docs.docker.com/get-docker/)
  - Verifica la instalación con `docker --version`.

- **Go (versión 1.22+)**: Lenguaje de programación utilizado para el desarrollo de los microservicios.
  - [Instrucciones de instalación](https://golang.org/doc/install)
  - Verifica la instalación con `go version`.

- **Terraform CLI**: Herramienta para la gestión de infraestructura como código.
  - [Instrucciones de instalación](https://learn.hashicorp.com/tutorials/terraform/install-cli)
  - Verifica la instalación con `terraform version`.

- **Cuenta de Twilio**: Plataforma de comunicaciones en la nube para la gestión de llamadas telefónicas.
  - [Registro en Twilio](https://www.twilio.com/try-twilio)
  - Necesitarás un número de teléfono de Twilio y las credenciales de API (SID y Auth Token).

## Sección 2: Configuración de GCP (Manual y Preparación para IaC)

### Creación de Proyecto GCP

1. Accede a la [Consola de Google Cloud](https://console.cloud.google.com/).
2. Haz clic en el selector de proyectos en la parte superior de la página.
3. Haz clic en "Nuevo proyecto".
4. Ingresa un nombre para tu proyecto (por ejemplo, "kairosia-prod").
5. Selecciona una organización y una ubicación de facturación, si corresponde.
6. Haz clic en "Crear".
7. Anota el ID del proyecto, lo necesitarás más adelante.

### Habilitación de APIs de GCP

Ejecuta los siguientes comandos para habilitar todas las APIs necesarias:

```bash
gcloud services enable compute.googleapis.com
gcloud services enable containerregistry.googleapis.com
gcloud services enable run.googleapis.com
gcloud services enable dialogflow.googleapis.com
gcloud services enable speech.googleapis.com
gcloud services enable texttospeech.googleapis.com
gcloud services enable firestore.googleapis.com
gcloud services enable bigquery.googleapis.com
gcloud services enable aiplatform.googleapis.com
gcloud services enable iam.googleapis.com
```

### Configuración Detallada de Dialogflow CX

1. **Crear un Agente de Dialogflow CX**:
   - Accede a la [Consola de Dialogflow CX](https://dialogflow.cloud.google.com/cx/).
   - Haz clic en "Crear agente".
   - Ingresa un nombre para el agente (por ejemplo, "KairosIA-Agent").
   - Selecciona la región (preferiblemente la misma región donde desplegarás los servicios).
   - Selecciona el proyecto de GCP que creaste anteriormente.
   - Haz clic en "Crear".
   - Anota el ID del agente, lo necesitarás más adelante.

2. **Configurar Flows e Intents**:
   - En la consola de Dialogflow CX, selecciona tu agente.
   - Crea al menos un flujo principal (por ejemplo, "Main Flow").
   - Dentro del flujo, crea intents básicos para la atención al cliente de telecomunicaciones:
     - `welcome`: Para saludar al usuario.
     - `billing_inquiry`: Para consultas de facturación.
     - `technical_support`: Para soporte técnico.
     - `service_change`: Para cambios en el servicio.
     - `agent_handoff`: Para solicitar hablar con un agente humano.
   - Para cada intent, configura frases de entrenamiento relevantes.
   - Anota los IDs de los flows e intents creados.

3. **Configurar el Fulfillment Webhook**:
   - En la consola de Dialogflow CX, ve a "Manage" > "Webhooks".
   - Haz clic en "Create".
   - Ingresa un nombre para el webhook (por ejemplo, "KairosIA-Webhook").
   - En el campo "URL", ingresa la URL del servicio de orquestación de voz (que obtendrás después del despliegue con Terraform).
   - Configura el método como "POST".
   - En "Request Format", selecciona "Dialogflow CX Webhook Format".
   - Haz clic en "Save".

4. **Configurar Custom Payload para Handoff a Agente Humano**:
   - En la consola de Dialogflow CX, ve al flujo donde deseas configurar el handoff.
   - Crea o selecciona una página para el handoff (por ejemplo, "Agent Handoff Page").
   - En la sección "Fulfillment", agrega un Custom Payload con el siguiente formato JSON:

```json
{
  "action": "LiveAgentHandoff",
  "transferNumber": "+56912345678",
  "reason": "El cliente ha solicitado hablar con un agente humano",
  "preserveContext": true
}
```

### Configuración de BigQuery

1. **Crear un Dataset**:
   - Accede a la [Consola de BigQuery](https://console.cloud.google.com/bigquery).
   - En el panel de navegación, selecciona tu proyecto.
   - Haz clic en "Crear dataset".
   - Ingresa un ID para el dataset (por ejemplo, "kairosia_conversations").
   - Selecciona una ubicación (preferiblemente la misma región donde desplegarás los servicios).
   - Haz clic en "Crear dataset".

2. **Crear la Tabla `conversation_transcripts`**:
   - En el panel de navegación, selecciona el dataset que creaste.
   - Haz clic en "Crear tabla".
   - En "Nombre de la tabla", ingresa "conversation_transcripts".
   - En "Esquema", define el siguiente esquema:

```
call_sid:STRING,
tenant_id:STRING,
from_number:STRING,
to_number:STRING,
start_timestamp:TIMESTAMP,
end_timestamp:TIMESTAMP,
duration_seconds:INTEGER,
transcript_entries:RECORD REPEATED
  - speaker:STRING
  - text:STRING
  - timestamp:TIMESTAMP
  - confidence:FLOAT
  - embedding:FLOAT REPEATED
dialogflow_metadata:RECORD
  - session_id:STRING
  - flow_id:STRING
  - intent_name:STRING
  - intent_confidence:FLOAT
  - parameters:STRING
  - page_id:STRING
handoff_occurred:BOOLEAN,
handoff_reason:STRING,
handoff_timestamp:TIMESTAMP,
embedding:FLOAT REPEATED,
created_at:TIMESTAMP
```

   - Haz clic en "Crear tabla".

### Configuración de Vertex AI Vector Search

1. **Generación de Embeddings en Lote**:
   - Para un MVP inicial, puedes generar embeddings de ejemplo utilizando el modelo `textembedding-gecko` de Vertex AI.
   - Estos embeddings servirán como datos iniciales para tu índice vectorial.

2. **Crear y Desplegar un Índice Vectorial**:
   - Accede a la [Consola de Vertex AI](https://console.cloud.google.com/vertex-ai).
   - Ve a "Vector Search" > "Índices".
   - Haz clic en "Crear".
   - Ingresa un nombre para el índice (por ejemplo, "kairosia-conversation-index").
   - Selecciona el tipo de índice "Vector Search".
   - Configura las dimensiones del vector según el modelo de embedding utilizado (por ejemplo, 768 para `textembedding-gecko`).
   - Configura los metadatos para filtrado (`tenant_id`, `from_number`).
   - Haz clic en "Crear".

3. **Importancia de los Filtros de Metadatos**:
   - Los filtros de metadatos (`tenant_id`, `from_number`) son cruciales para la búsqueda contextual.
   - Permiten recuperar conversaciones relevantes para un cliente o empresa específica.
   - Mejoran la precisión y personalización de las respuestas de la IA.

## Sección 3: Variables de Entorno (.env)

El archivo `.env.example` incluye todas las variables de entorno necesarias para configurar los servicios y Terraform. A continuación, se explica cada variable:

### Variables Generales
- `GCP_PROJECT_ID`: ID del proyecto de GCP.
- `GCP_REGION`: Región de GCP donde se desplegarán los servicios.
- `GCP_ZONE`: Zona específica dentro de la región de GCP.

### Variables de Twilio
- `TWILIO_ACCOUNT_SID`: SID de la cuenta de Twilio.
- `TWILIO_AUTH_TOKEN`: Token de autenticación de Twilio.
- `TWILIO_PHONE_NUMBER`: Número de teléfono de Twilio asignado a tu cuenta.

### Variables de Dialogflow CX
- `DIALOGFLOW_AGENT_ID`: ID del agente de Dialogflow CX.
- `DIALOGFLOW_LOCATION`: Ubicación del agente de Dialogflow CX.
- `DIALOGFLOW_DEFAULT_LANGUAGE_CODE`: Código de idioma predeterminado para Dialogflow CX (por ejemplo, "es-CL" para español de Chile).

### Variables de Google Cloud Speech-to-Text
- `STT_LANGUAGE_CODE`: Código de idioma para Speech-to-Text (por ejemplo, "es-CL").
- `STT_MODEL`: Modelo de Speech-to-Text a utilizar (por ejemplo, "phone_call").

### Variables de Google Cloud Text-to-Speech
- `TTS_LANGUAGE_CODE`: Código de idioma para Text-to-Speech (por ejemplo, "es-CL").
- `TTS_VOICE_NAME`: Nombre de la voz para Text-to-Speech (por ejemplo, "es-CL-Standard-A").
- `TTS_SPEAKING_RATE`: Velocidad de habla para Text-to-Speech (por ejemplo, "1.0").

### Variables de Firestore
- `FIRESTORE_COLLECTION`: Nombre de la colección de Firestore para almacenar el estado de las conversaciones.

### Variables de BigQuery
- `BIGQUERY_DATASET`: Nombre del dataset de BigQuery.
- `BIGQUERY_TABLE`: Nombre de la tabla de BigQuery para almacenar las transcripciones.

### Variables de Vertex AI
- `VERTEX_AI_EMBEDDING_MODEL`: Modelo de embedding de Vertex AI a utilizar (por ejemplo, "textembedding-gecko").
- `VERTEX_AI_VECTOR_SEARCH_INDEX`: ID del índice de Vector Search.
- `VERTEX_AI_VECTOR_SEARCH_ENDPOINT`: ID del endpoint de Vector Search.

### Variables de Terraform
- `TF_VAR_project_id`: ID del proyecto de GCP para Terraform.
- `TF_VAR_region`: Región de GCP para Terraform.
- `TF_VAR_zone`: Zona de GCP para Terraform.
- `TF_VAR_service_account_suffix`: Sufijo para las cuentas de servicio creadas por Terraform.

## Sección 4: Despliegue con Terraform (Infraestructura como Código)

### Configuración de Terraform

1. **Inicializar Terraform**:
   - Navega al directorio `terraform` del proyecto.
   - Ejecuta `terraform init` para inicializar Terraform.

2. **Configurar Variables de Terraform**:
   - Crea un archivo `terraform.tfvars` basado en las variables definidas en `variables.tf`.
   - Alternativamente, puedes utilizar las variables de entorno definidas en el archivo `.env`.

3. **Planificar el Despliegue**:
   - Ejecuta `terraform plan` para ver los recursos que se crearán.
   - Verifica que los recursos planificados coincidan con tus expectativas.

4. **Aplicar el Despliegue**:
   - Ejecuta `terraform apply` para crear los recursos en GCP.
   - Confirma la creación de recursos cuando se te solicite.

### Recursos Aprovisionados por Terraform

El código Terraform provisto aprovisiona los siguientes recursos:

- **2 Servicios de Cloud Run**:
  - `voice-orchestration-service`: Servicio principal para la orquestación de voz.
  - `conversation-history-service`: Servicio para el almacenamiento del historial de conversaciones.

- **2 Cuentas de Servicio (Service Accounts)**:
  - `voice-orchestration-sa`: Cuenta de servicio para el servicio de orquestación de voz.
  - `history-service-sa`: Cuenta de servicio para el servicio de historial de conversaciones.

- **Roles IAM**:
  - `roles/dialogflow.client`: Para interactuar con Dialogflow CX.
  - `roles/speech.user`: Para utilizar Speech-to-Text.
  - `roles/texttospeech.user`: Para utilizar Text-to-Speech.
  - `roles/firestore.user`: Para interactuar con Firestore.
  - `roles/bigquery.dataEditor`: Para escribir datos en BigQuery.
  - `roles/aiplatform.user`: Para utilizar Vertex AI.
  - `roles/run.invoker`: Para la comunicación entre servicios.

- **Dataset de BigQuery**:
  - Dataset para almacenar las transcripciones de las conversaciones.

- **Tabla de BigQuery**:
  - Tabla `conversation_transcripts` con el esquema completo para almacenar las transcripciones.

- **Base de Datos de Firestore**:
  - Base de datos en modo Native para almacenar el estado de las conversaciones.

- **Referencia Conceptual a Vertex AI Vector Search**:
  - Aunque la creación inicial de datos para el índice suele ser un proceso fuera de Terraform para un MVP, se incluye una referencia conceptual a la creación y despliegue del índice y endpoint de Vector Search.

## Sección 5: Configuración de Twilio

Para configurar Twilio para que apunte a tu servicio de orquestación de voz en Cloud Run, sigue estos pasos:

1. **Accede a la [Consola de Twilio](https://www.twilio.com/console)**.
2. **Navega a "Phone Numbers" > "Manage" > "Active Numbers"**.
3. **Selecciona el número de teléfono que deseas configurar**.
4. **En la sección "Voice & Fax", configura lo siguiente**:
   - En "A Call Comes In", selecciona "Webhook".
   - En el campo de URL, ingresa la URL del servicio de orquestación de voz en Cloud Run (obtenida de las salidas de Terraform).
   - Asegúrate de que el método esté configurado como "HTTP POST".
   - Haz clic en "Save".

5. **Verifica la configuración**:
   - Realiza una llamada de prueba al número de Twilio.
   - Verifica que la llamada sea recibida por el servicio de orquestación de voz.

## Sección 6: Prueba del Sistema

Para probar el sistema completo, sigue estos pasos:

1. **Realiza una Llamada de Prueba**:
   - Llama al número de Twilio configurado.
   - Interactúa con el sistema de voz siguiendo las indicaciones.

2. **Verifica los Logs**:
   - Accede a la [Consola de Cloud Run](https://console.cloud.google.com/run).
   - Selecciona el servicio `voice-orchestration-service`.
   - Haz clic en "Logs" para ver los logs del servicio.
   - Verifica que los logs muestren la recepción de la llamada y la interacción con Dialogflow CX.

3. **Verifica los Datos en BigQuery**:
   - Accede a la [Consola de BigQuery](https://console.cloud.google.com/bigquery).
   - Ejecuta una consulta para verificar que los datos de la conversación se hayan almacenado correctamente:

```sql
SELECT * FROM `your-project-id.kairosia_conversations.conversation_transcripts`
ORDER BY start_timestamp DESC
LIMIT 10;
```

4. **Verifica el Estado en Firestore**:
   - Accede a la [Consola de Firestore](https://console.cloud.google.com/firestore).
   - Navega a la colección configurada para almacenar el estado de las conversaciones.
   - Verifica que se haya creado un documento para la llamada de prueba.

## Sección 7: Consideraciones para Producción

Antes de desplegar el sistema en un entorno de producción, considera las siguientes recomendaciones:

### Seguridad
- **Autenticación de Cloud Run**: Elimina la opción `allow-unauthenticated` y configura la autenticación adecuada para los servicios de Cloud Run.
- **Secretos**: Utiliza Secret Manager para almacenar y gestionar secretos como tokens de API.
- **IAM**: Refina los permisos IAM para seguir el principio de privilegio mínimo.

### Observabilidad
- **Logging**: Implementa logging estructurado para facilitar el análisis y la depuración.
- **Monitoring**: Configura alertas y dashboards en Cloud Monitoring para supervisar el rendimiento y la disponibilidad del sistema.
- **Tracing**: Implementa tracing distribuido para identificar cuellos de botella y optimizar el rendimiento.

### CI/CD
- **Pipeline de CI/CD**: Implementa un pipeline de CI/CD para automatizar el proceso de construcción, prueba y despliegue.
- **Entornos**: Configura entornos separados para desarrollo, pruebas y producción.
- **Versionado**: Implementa un sistema de versionado para los servicios y la infraestructura.

### Escalabilidad
- **Autoscaling**: Configura el autoscaling de Cloud Run para manejar picos de tráfico.
- **Cuotas**: Monitorea y ajusta las cuotas de API para evitar limitaciones.
- **Caché**: Implementa estrategias de caché para reducir la latencia y mejorar el rendimiento.

### Resiliencia
- **Retry Logic**: Implementa lógica de reintentos para manejar fallos transitorios.
- **Circuit Breaker**: Implementa el patrón de circuit breaker para evitar cascadas de fallos.
- **Backup**: Configura backups regulares de los datos críticos.

Con estas consideraciones en mente, estarás mejor preparado para desplegar KairosIA en un entorno de producción robusto y escalable.
