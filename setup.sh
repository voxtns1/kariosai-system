#!/bin/bash

# Script para configurar y ejecutar KairosIA localmente

# Cargar variables de entorno
if [ -f .env ]; then
  export $(cat .env | grep -v '#' | awk '/=/ {print $1}')
else
  echo "Archivo .env no encontrado. Creando desde .env.example..."
  cp .env.example .env
  echo "Por favor, edita el archivo .env con tus valores antes de continuar."
  exit 1
fi

# Función para verificar dependencias
check_dependencies() {
  echo "Verificando dependencias..."
  
  # Verificar Go
  if ! command -v go &> /dev/null; then
    echo "Go no está instalado. Por favor, instala Go 1.22 o superior."
    exit 1
  fi
  
  # Verificar Docker
  if ! command -v docker &> /dev/null; then
    echo "Docker no está instalado. Se recomienda para ejecutar en contenedores."
    read -p "¿Deseas continuar sin Docker? (s/n): " continue_without_docker
    if [ "$continue_without_docker" != "s" ]; then
      exit 1
    fi
  fi
  
  # Verificar gcloud CLI
  if ! command -v gcloud &> /dev/null; then
    echo "Google Cloud SDK no está instalado. Se requiere para despliegue en GCP."
    read -p "¿Deseas continuar sin gcloud? (s/n): " continue_without_gcloud
    if [ "$continue_without_gcloud" != "s" ]; then
      exit 1
    fi
  fi
  
  # Verificar Terraform
  if ! command -v terraform &> /dev/null; then
    echo "Terraform no está instalado. Se requiere para infraestructura como código."
    read -p "¿Deseas continuar sin Terraform? (s/n): " continue_without_terraform
    if [ "$continue_without_terraform" != "s" ]; then
      exit 1
    fi
  fi
  
  echo "Todas las dependencias están disponibles o han sido confirmadas."
}

# Función para configurar el proyecto
setup_project() {
  echo "Configurando el proyecto..."
  
  # Descargar dependencias de Go
  echo "Descargando dependencias de Go..."
  go mod download
  cd voice-orchestration-service && go mod download && cd ..
  cd conversation-history-service && go mod download && cd ..
  
  echo "Proyecto configurado correctamente."
}

# Función para construir los servicios
build_services() {
  echo "Construyendo servicios..."
  
  # Crear directorios bin si no existen
  mkdir -p voice-orchestration-service/bin
  mkdir -p conversation-history-service/bin
  
  # Construir servicio de orquestación de voz
  echo "Construyendo servicio de orquestación de voz..."
  cd voice-orchestration-service && go build -o bin/voice-orchestration-service && cd ..
  
  # Construir servicio de historial de conversaciones
  echo "Construyendo servicio de historial de conversaciones..."
  cd conversation-history-service && go build -o bin/conversation-history-service && cd ..
  
  echo "Servicios construidos correctamente."
}

# Función para ejecutar los servicios localmente
run_local() {
  echo "Ejecutando servicios localmente..."
  
  # Ejecutar servicio de historial de conversaciones en segundo plano
  cd conversation-history-service && go run main.go &
  HISTORY_PID=$!
  cd ..
  
  # Ejecutar servicio de orquestación de voz en primer plano
  cd voice-orchestration-service && go run main.go
  
  # Matar el proceso del servicio de historial al terminar
  kill $HISTORY_PID
}

# Función para construir imágenes Docker
build_docker() {
  echo "Construyendo imágenes Docker..."
  
  # Construir imagen para el servicio de orquestación de voz
  echo "Construyendo imagen para el servicio de orquestación de voz..."
  docker build -t kairosia-voice-orchestration-service ./voice-orchestration-service
  
  # Construir imagen para el servicio de historial de conversaciones
  echo "Construyendo imagen para el servicio de historial de conversaciones..."
  docker build -t kairosia-conversation-history-service ./conversation-history-service
  
  echo "Imágenes Docker construidas correctamente."
}

# Función para ejecutar servicios en Docker
run_docker() {
  echo "Ejecutando servicios en Docker..."
  
  # Ejecutar servicio de historial de conversaciones
  echo "Ejecutando servicio de historial de conversaciones en Docker..."
  docker run -d -p 8081:8080 --env-file .env --name kairosia-history kairosia-conversation-history-service
  
  # Ejecutar servicio de orquestación de voz
  echo "Ejecutando servicio de orquestación de voz en Docker..."
  docker run -p 8080:8080 --env-file .env --name kairosia-voice kairosia-voice-orchestration-service
  
  # Limpiar contenedores al terminar
  docker rm -f kairosia-history kairosia-voice
}

# Función para configurar GCP
setup_gcp() {
  echo "Configurando Google Cloud Platform..."
  
  # Autenticarse en GCP
  gcloud auth login
  
  # Configurar proyecto
  gcloud config set project $GCP_PROJECT_ID
  
  # Habilitar APIs necesarias
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
  
  echo "Google Cloud Platform configurado correctamente."
}

# Función para construir y publicar imágenes en GCP
build_push_gcp() {
  echo "Construyendo y publicando imágenes en Google Cloud Platform..."
  
  # Construir y publicar imagen para el servicio de orquestación de voz
  echo "Construyendo y publicando imagen para el servicio de orquestación de voz..."
  gcloud builds submit --tag gcr.io/$GCP_PROJECT_ID/voice-orchestration-service:latest ./voice-orchestration-service
  
  # Construir y publicar imagen para el servicio de historial de conversaciones
  echo "Construyendo y publicando imagen para el servicio de historial de conversaciones..."
  gcloud builds submit --tag gcr.io/$GCP_PROJECT_ID/conversation-history-service:latest ./conversation-history-service
  
  echo "Imágenes construidas y publicadas correctamente en Google Cloud Platform."
}

# Función para desplegar servicios en GCP
deploy_gcp() {
  echo "Desplegando servicios en Google Cloud Platform..."
  
  # Desplegar servicio de historial de conversaciones
  echo "Desplegando servicio de historial de conversaciones..."
  gcloud run deploy conversation-history-service \
    --image gcr.io/$GCP_PROJECT_ID/conversation-history-service:latest \
    --platform managed \
    --region $GCP_REGION \
    --allow-unauthenticated
  
  # Obtener URL del servicio de historial de conversaciones
  HISTORY_URL=$(gcloud run services describe conversation-history-service --platform managed --region $GCP_REGION --format 'value(status.url)')
  
  # Actualizar variable de entorno para el servicio de orquestación de voz
  export CONVERSATION_HISTORY_SERVICE_URL=$HISTORY_URL
  
  # Desplegar servicio de orquestación de voz
  echo "Desplegando servicio de orquestación de voz..."
  gcloud run deploy voice-orchestration-service \
    --image gcr.io/$GCP_PROJECT_ID/voice-orchestration-service:latest \
    --platform managed \
    --region $GCP_REGION \
    --allow-unauthenticated \
    --set-env-vars CONVERSATION_HISTORY_SERVICE_URL=$HISTORY_URL
  
  # Obtener URL del servicio de orquestación de voz
  VOICE_URL=$(gcloud run services describe voice-orchestration-service --platform managed --region $GCP_REGION --format 'value(status.url)')
  
  echo "Servicios desplegados correctamente en Google Cloud Platform."
  echo "URL del servicio de orquestación de voz: $VOICE_URL"
  echo "URL del servicio de historial de conversaciones: $HISTORY_URL"
}

# Función para desplegar con Terraform
deploy_terraform() {
  echo "Desplegando infraestructura con Terraform..."
  
  # Inicializar Terraform
  cd terraform && terraform init
  
  # Planificar despliegue
  terraform plan
  
  # Confirmar despliegue
  read -p "¿Deseas continuar con el despliegue? (s/n): " continue_deploy
  if [ "$continue_deploy" != "s" ]; then
    echo "Despliegue cancelado."
    cd ..
    return
  fi
  
  # Aplicar despliegue
  terraform apply -auto-approve
  
  # Obtener salidas
  VOICE_URL=$(terraform output -raw voice_orchestration_service_url)
  HISTORY_URL=$(terraform output -raw conversation_history_service_url)
  
  cd ..
  
  echo "Infraestructura desplegada correctamente con Terraform."
  echo "URL del servicio de orquestación de voz: $VOICE_URL"
  echo "URL del servicio de historial de conversaciones: $HISTORY_URL"
}

# Menú principal
show_menu() {
  echo "=== KairosIA - Sistema de Atención Telefónica Conversacional ==="
  echo "1. Verificar dependencias"
  echo "2. Configurar proyecto"
  echo "3. Construir servicios"
  echo "4. Ejecutar servicios localmente"
  echo "5. Construir imágenes Docker"
  echo "6. Ejecutar servicios en Docker"
  echo "7. Configurar Google Cloud Platform"
  echo "8. Construir y publicar imágenes en GCP"
  echo "9. Desplegar servicios en GCP (manual)"
  echo "10. Desplegar infraestructura con Terraform"
  echo "0. Salir"
  echo "=============================================================="
  read -p "Selecciona una opción: " option
  
  case $option in
    1) check_dependencies ;;
    2) setup_project ;;
    3) build_services ;;
    4) run_local ;;
    5) build_docker ;;
    6) run_docker ;;
    7) setup_gcp ;;
    8) build_push_gcp ;;
    9) deploy_gcp ;;
    10) deploy_terraform ;;
    0) exit 0 ;;
    *) echo "Opción inválida" ;;
  esac
  
  read -p "Presiona Enter para continuar..."
  show_menu
}

# Iniciar el script
show_menu
