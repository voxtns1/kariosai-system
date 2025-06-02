package utils

import (
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/bigquery"
	"google.golang.org/protobuf/types/known/structpb"
)

// GetEnv obtiene una variable de entorno con un valor predeterminado
func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Atoi convierte una cadena a entero con un valor predeterminado
func Atoi(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return i
}

// ParseFloat convierte una cadena a float64 con un valor predeterminado
func ParseFloat(s string, defaultValue float64) float64 {
	if s == "" {
		return defaultValue
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return defaultValue
	}
	return f
}

// StructToProtoStruct convierte un mapa a un protobuf struct
func StructToProtoStruct(m map[string]interface{}) (*structpb.Struct, error) {
	return structpb.NewStruct(m)
}

// ProtoStructToMap convierte un protobuf struct a un mapa
func ProtoStructToMap(s *structpb.Struct) map[string]interface{} {
	if s == nil {
		return nil
	}
	m := make(map[string]interface{})
	for k, v := range s.Fields {
		m[k] = ProtoValueToInterface(v)
	}
	return m
}

// ProtoValueToInterface convierte un protobuf value a una interfaz
func ProtoValueToInterface(v *structpb.Value) interface{} {
	switch x := v.Kind.(type) {
	case *structpb.Value_NullValue:
		return nil
	case *structpb.Value_NumberValue:
		return v.GetNumberValue()
	case *structpb.Value_StringValue:
		return v.GetStringValue()
	case *structpb.Value_BoolValue:
		return v.GetBoolValue()
	case *structpb.Value_StructValue:
		return ProtoStructToMap(v.GetStructValue())
	case *structpb.Value_ListValue:
		list := v.GetListValue()
		result := make([]interface{}, len(list.Values))
		for i, v := range list.Values {
			result[i] = ProtoValueToInterface(v)
		}
		return result
	default:
		return nil
	}
}

// StructToProtoValue convierte una estructura a un protobuf value
func StructToProtoValue(i interface{}) (*structpb.Value, error) {
	return structpb.NewValue(i)
}

// MapToBigQueryValue convierte un mapa a un valor de BigQuery
func MapToBigQueryValue(m map[string]interface{}) (bigquery.Value, error) {
	return bigquery.Value(m), nil
}

// FormatPhoneNumber formatea un número de teléfono para asegurar el formato E.164
func FormatPhoneNumber(phone string) string {
	// Eliminar todos los caracteres no numéricos
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)

	// Si no comienza con +, agregar el prefijo +
	if !strings.HasPrefix(phone, "+") {
		// Si es un número chileno sin código de país, agregar +56
		if len(digits) == 9 && (digits[0] == '9' || digits[0] == '2') {
			return "+56" + digits
		}
		return "+" + digits
	}

	return phone
}

// TruncateString trunca una cadena a una longitud máxima
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// GenerateSessionID genera un ID de sesión único para Dialogflow CX
func GenerateSessionID(callSid string) string {
	return "twilio-" + callSid
}
