# 🎨 Penpot Manager

> Instalador interactivo por terminal para correr [Penpot](https://penpot.app) — la alternativa open source a Figma — en tu propia máquina usando Docker.

Sin configuraciones manuales, sin tocar YAML, sin dolor. Descargás el binario, lo ejecutás, y en minutos tenés Penpot corriendo localmente.

---

## ¿Qué es Penpot?

[Penpot](https://penpot.app) es una herramienta de diseño y prototipado open source, similar a Figma. Podés usarla para diseñar interfaces, crear prototipos y colaborar en equipo — pero alojada en tu propia infraestructura, sin depender de servicios de terceros.

---

## Características del instalador

- ✅ **Detección automática de OS** — Linux y Windows
- ✅ **Instalación de Docker** — si no lo tenés, te pregunta si instalarlo
- ✅ **Menú interactivo** — navegación con flechas, sin escribir comandos
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
# Descargar el binario
curl -fsSL https://github.com/estefrac/penpot-installer/releases/latest/download/penpot-manager-linux-amd64 -o penpot-manager

# Darle permisos de ejecución
chmod +x penpot-manager

# Ejecutar
./penpot-manager
```

> En ARM64 (Raspberry Pi, Apple Silicon con Linux, etc.):
> ```bash
> curl -fsSL https://github.com/estefrac/penpot-installer/releases/latest/download/penpot-manager-linux-arm64 -o penpot-manager
> chmod +x penpot-manager
> ./penpot-manager
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

Al ejecutar el instalador verás el menú principal. Las opciones disponibles cambian automáticamente según el estado actual de Penpot.

### Primera vez (instalación)

```
🎨 Penpot Manager

ℹ Sistema operativo detectado: linux
✔ Docker 28.5.2 detectado

? ¿Qué querés hacer?
  ❯ 🚀 Instalar Penpot
    ❌ Salir
```

El instalador te preguntará:
- **Directorio de instalación** (por defecto: `~/penpot`)
- **Puerto** (por defecto: `9001`)
- **Confirmación** antes de proceder

Luego descarga las imágenes de Docker e inicia los contenedores automáticamente.

### Una vez instalado

```
? ¿Qué querés hacer?
  ❯ ⏹️  Detener Penpot
    🌐 Abrir en navegador
    📊 Ver estado
    🔄 Actualizar Penpot
    🗑️  Desinstalar Penpot
    ❌ Salir
```

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
| 🔄 **Actualizar Penpot** | Baja las últimas imágenes y reinicia |
| 📊 **Ver estado** | Muestra el estado de cada contenedor |
| 🌐 **Abrir en navegador** | Abre `http://localhost:9001` directamente |
| 🗑️ **Desinstalar Penpot** | Elimina contenedores (con opción de borrar datos) |

---

## ¿Qué pasa con mis datos?

Penpot guarda todo en **volúmenes de Docker**. Eso significa que:

- **Detener Penpot** → tus proyectos y archivos están seguros
- **Actualizar Penpot** → tus datos se conservan
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
├── main.go                    ← Menú principal y flujo de la app
├── internal/
│   ├── ui/        ui.go       ← Banner, colores, spinners
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
| [pterm](https://github.com/pterm/pterm) | UI de terminal: colores, spinners, tablas |
| [survey](https://github.com/AlecAivazis/survey) | Prompts interactivos |
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
