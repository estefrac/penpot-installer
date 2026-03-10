package tui

// Mensajes (commands) de Bubble Tea para comunicar
// resultados de operaciones asincrónicas al modelo.

// msgDockerReady se envía cuando Docker está listo para usar
type msgDockerReady struct{}

// msgDockerError se envía cuando hay un error con Docker
type msgDockerError struct{ err error }

// msgOperationDone se envía cuando una operación completó con éxito
type msgOperationDone struct{ message string }

// msgOperationError se envía cuando una operación falló
type msgOperationError struct{ err error }

// msgStatusLoaded se envía con el estado de los contenedores
type msgStatusLoaded struct {
	isInstalled bool
	isRunning   bool
	containers  []containerInfo
}

// containerInfo info de un contenedor Docker
type containerInfo struct {
	name   string
	status string
	ports  string
}

// msgWindowSize se envía cuando cambia el tamaño de la terminal
// (ya viene de tea.WindowSizeMsg, pero lo re-exportamos por claridad)
