// Package claude implementa el servicio de IA generativa con la API de
// Claude, mediante el SDK oficial github.com/anthropics/anthropic-sdk-go.
//
// El modelo se configura con LLM_MODEL (por defecto claude-opus-5-5) y la
// credencial con ANTHROPIC_API_KEY. El prompt se construye solo con los
// insumos permitidos por RN-43 y pide una síntesis, no una evaluación (RN-46).
package claude
