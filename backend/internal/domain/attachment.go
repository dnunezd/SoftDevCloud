package domain

// TaskAttachment son los metadatos de un archivo adjunto a una tarea (RN-23,
// RF-06.4).
//
// El contenido binario no vive en la base de datos: se delega al puerto
// FileStorage y aquí solo se guarda la clave de almacenamiento, que genera el
// sistema y nunca el nombre del archivo del usuario. Pertenece al agregado de
// su tarea: al eliminarla, se eliminan sus adjuntos.
type TaskAttachment struct {
	ID          string
	TaskID      string
	FileName    string
	ContentType string
	SizeBytes   int64
	StorageKey  string
}
