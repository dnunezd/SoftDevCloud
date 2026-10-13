// Package ports define las interfaces que la capa de aplicación necesita del
// exterior (puertos de salida).
//
// Son las "costuras" de evolución del proyecto: cada entrega cambia o agrega
// adaptadores que implementan estos puertos, sin tocar dominio ni casos de uso.
package ports
