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

// msgDockerNotInstalled se envía cuando Docker no está instalado
type msgDockerNotInstalled struct{ os string }

// msgDockerNotRunning se envía cuando Docker está instalado pero no corriendo
type msgDockerNotRunning struct{ os string }

// msgDockerWindowsStarting se envía cuando Docker Desktop fue lanzado y se está esperando que arranque
type msgDockerWindowsStarting struct{}

// msgLogLine se envía con cada línea de output de un comando en streaming.
// closed=true indica que el canal se cerró (no hay más líneas).
type msgLogLine struct {
	line   string
	closed bool
}

// msgDockerInstallDone se envía cuando la instalación de Docker completó
type msgDockerInstallDone struct{}

// msgDockerInstallError se envía cuando la instalación de Docker falló
type msgDockerInstallError struct{ err error }

// msgChangeView solicita al modelo principal cambiar la vista activa
type msgChangeView struct{ newView view }

// msgDockerComposeNotInstalled se envía cuando docker compose (V2) no está disponible
// pero Docker sí está instalado y corriendo.
type msgDockerComposeNotInstalled struct{ os string }
