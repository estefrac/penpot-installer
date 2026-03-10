# 🎨 Penpot Manager

> Instalador interactivo por terminal para correr [Penpot](https://penpot.app) — la alternativa open source a Figma — en tu propia máquina usando Docker.

Sin configuraciones manuales, sin tocar YAML, sin dolor. Descargás el binario, lo ejecutás, y en minutos tenés Penpot corriendo localmente.

---

## ¿Qué es Penpot?

[Penpot](https://penpot.app) es una herramienta de diseño y prototipado open source, similar a Figma. Podés usarla para diseñar interfaces, crear prototipos y colaborar en equipo — pero alojada en tu propia infraestructura, sin depender de servicios de terceros.

---

## Características del instalador

- ✅ **TUI profesional** — interfaz de terminal con paneles, colores y navegación por teclado
- ✅ **Detección automática de OS** — Linux y Windows
- ✅ **Instalación de Docker** — si no lo tenés, te lo indica con instrucciones claras
- ✅ **Menú interactivo** — navegación con ↑↓, sin escribir comandos
- ✅ **Configuración guiada** — directorio de instalación y puerto personalizables
- ✅ **Gestión completa** — instalar, iniciar, detener, actualizar y desinstalar
- ✅ **Sin dependencias** — un solo binario, nada más

---

## Requisitos

| Requisito | Detalle |
|---|---|
| **OS** | Linux (x64 / ARM64) o Windows 10/11 (x64) |
| **Docker** | El instalador te ayuda a instalarlo si no lo tenés |
| **Conexión a internet** | Para descargar las imágenes de Docker |
| **RAM** | Mínimo 4 GB recomendados |
| **Disco** | ~3 GB para las imágenes de Docker |

> **En Linux:** necesitás permisos de administrador (`sudo`) para instalar Docker y gestionar servicios.

---

## Instalación

### Linux

```bash
# Descargar el binario, darle permisos y ejecutar
curl -fsSL https://github.com/estefrac/penpot-installer/releases/latest/download/penpot-manager-linux-amd64 -o /tmp/penpot-manager && chmod +x /tmp/penpot-manager && /tmp/penpot-manager
```

> En ARM64 (Raspberry Pi, Apple Silicon con Linux, etc.):
> ```bash
> curl -fsSL https://github.com/estefrac/penpot-installer/releases/latest/download/penpot-manager-linux-arm64 -o /tmp/penpot-manager && chmod +x /tmp/penpot-manager && /tmp/penpot-manager
> ```

### Windows

1. Descargá el archivo `penpot-manager-windows-amd64.exe` desde la [página de Releases](https://github.com/estefrac/penpot-installer/releases/latest)
2. Abrí una terminal (CMD o PowerShell) **como administrador**
3. Navegá a la carpeta donde descargaste el archivo
4. Ejecutá:
   ```cmd
   penpot-manager-windows-amd64.exe
   ```

---

## Uso

Al ejecutar el instalador verás el TUI completo con paneles. Las opciones disponibles cambian automáticamente según el estado actual de Penpot.

### Navegación

| Tecla | Acción |
|---|---|
| `↑` / `↓` | Moverse por el menú |
| `Enter` | Seleccionar opción |
| `Tab` | Siguiente campo (en formularios) |
| `s` / `y` | Confirmar acción |
| `n` | Cancelar acción |
| `Esc` | Volver al menú |
| `q` | Salir |

### Primera vez (instalación)

Al no estar instalado, el menú muestra solo **Instalar Penpot**. Al seleccionarlo, un formulario te pedirá:
- **Directorio de instalación** (por defecto: `~/penpot`)
- **Puerto** (por defecto: `9001`)
- **Confirmación** antes de proceder

Luego un spinner animado muestra el progreso mientras Docker descarga las imágenes (~2-3 GB). Cuando termina, aparece un mensaje de confirmación.

> ⏱️ La primera instalación puede tardar entre 5 y 15 minutos dependiendo de tu conexión.

### Una vez instalado

El panel izquierdo muestra las opciones disponibles según si Penpot está corriendo o detenido. El panel derecho muestra el estado en tiempo real, el directorio y la URL de acceso.

### Acceder a Penpot

Una vez instalado, abrí tu navegador en:

```
http://localhost:9001
```

> Si configuraste un puerto distinto, usá ese número en lugar de `9001`.

La primera vez que accedas tendrás que **crear una cuenta** (es local, no va a ningún servidor externo).

---

## Opciones del menú

| Opción | Descripción |
|---|---|
| 🚀 **Instalar Penpot** | Descarga imágenes y levanta los contenedores |
| ▶️ **Iniciar Penpot** | Inicia los contenedores si están detenidos |
| ⏹️ **Detener Penpot** | Detiene los contenedores (los datos se conservan) |
| 🔄 **Actualizar Penpot** | Actualiza el `docker-compose.yaml` oficial, baja las últimas imágenes y reinicia |
| 📊 **Ver estado** | Muestra el estado de cada contenedor |
| 🌐 **Abrir en navegador** | Abre `http://localhost:9001` directamente |
| 🗑️ **Desinstalar Penpot** | Elimina contenedores (con opción de borrar datos) |

---

## ¿Qué pasa con mis datos?

Penpot guarda todo en **volúmenes de Docker**. Eso significa que:

- **Detener Penpot** → tus proyectos y archivos están seguros
- **Actualizar Penpot** → refresca la configuración oficial y las imágenes, tus datos se conservan
- **Desinstalar sin borrar datos** → los volúmenes quedan en Docker, podés reinstalar y recuperarlos
- **Desinstalar borrando datos** → ⚠️ esto es irreversible, el instalador te advierte y pide doble confirmación

---

## Compilar desde el código fuente

Si preferís compilarlo vos mismo:

```bash
# Requisitos: Go 1.24+
git clone https://github.com/estefrac/penpot-installer.git
cd penpot-installer

# Para Linux
go build -o penpot-manager .

# Para Windows (cross-compile desde Linux/Mac)
GOOS=windows GOARCH=amd64 go build -o penpot-manager.exe .
```

---

## Estructura del proyecto

```
penpot-installer/
├── main.go                    ← Entry point — arranca el TUI
├── internal/
│   ├── tui/                   ← TUI completo (Bubble Tea)
│   │   ├── model.go           ← Modelo principal: vistas, navegación, operaciones
│   │   ├── styles.go          ← Paleta de colores Penpot + estilos Lip Gloss
│   │   ├── banner.go          ← ASCII art con gradiente
│   │   └── messages.go        ← Tipos de mensajes para operaciones async
│   ├── system/    system.go   ← Detección de OS, ejecutar comandos
│   ├── docker/    docker.go   ← Verificar e instalar Docker
│   └── penpot/    penpot.go   ← Instalar, actualizar y desinstalar Penpot
└── .github/
    └── workflows/
        └── release.yml        ← CI/CD: compila binarios en cada release
```

---

## Tecnologías usadas

| Librería | Uso |
|---|---|
| [Bubble Tea](https://github.com/charmbracelet/bubbletea) | Framework TUI (patrón Model/Update/View) |
| [Lip Gloss](https://github.com/charmbracelet/lipgloss) | Estilos CSS-like para terminal |
| [Bubbles](https://github.com/charmbracelet/bubbles) | Componentes: textinput, spinner |
| Go stdlib | Detección de OS, ejecución de comandos, manejo de archivos |

---

## Preguntas frecuentes

**¿Puedo usar Penpot en red local para que otros lo usen?**
Sí. Reemplazá `localhost` por la IP de tu máquina. Asegurate de que el puerto esté abierto en el firewall.

**¿Funciona en Mac?**
El instalador detecta macOS pero la instalación automática de Docker no está disponible todavía. Instalá [Docker Desktop para Mac](https://www.docker.com/products/docker-desktop/) manualmente y luego ejecutá el instalador.

**¿Cuánto tarda la primera instalación?**
Depende de tu conexión. Las imágenes de Docker pesan aproximadamente 2-3 GB. Con buena conexión, unos 5-10 minutos.

**¿Es seguro? ¿Manda datos a algún servidor?**
Todo corre localmente en tu máquina. El instalador solo descarga imágenes desde Docker Hub y el `docker-compose.yaml` oficial de Penpot.

---

## Contribuir

1. Hacé fork del repo
2. Creá una rama: `git checkout -b feature/mi-mejora`
3. Hacé tus cambios y commiteá: `git commit -m 'feat: descripción'`
4. Pusheá: `git push origin feature/mi-mejora`
5. Abrí un Pull Request

---

## Licencia

MIT — hacé lo que quieras con esto.

---

> Hecho con ❤️ para que instalar Penpot sea tan fácil como debería ser.
