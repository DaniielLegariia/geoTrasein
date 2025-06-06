# GeoTrasein

Proyecto para procesar datos de geocercas desde Oracle.

## Requisitos

- Go 1.16 o superior
- Oracle Client
- Variables de entorno configuradas

## Variables de Entorno

Crear un archivo `.env` en la raíz del proyecto con las siguientes variables:

```env
ORACLE_USER=tu_usuario
ORACLE_PASSWORD=tu_password
ORACLE_HOST=tu_host
ORACLE_PORT=tu_puerto
ORACLE_SERVICE=tu_servicio
```

## Instalación

1. Clonar el repositorio:
```bash
git clone https://github.com/tu-usuario/geoTrasein.git
cd geoTrasein
```

2. Instalar dependencias:
```bash
go mod download
```

3. Compilar el proyecto:
```bash
go build -o geoTrasein cmd/parser/main.go
```

## Uso

Ejecutar el programa:
```bash
./geoTrasein
```

El programa se conectará a Oracle, ejecutará el procedimiento almacenado `prc_ws_nestle_get_evento_geo_trasein` y procesará los datos de geocercas.

## Estructura del Proyecto

```
.
├── cmd/
│   └── parser/
│       └── main.go
├── pkg/
│   ├── data/
│   │   └── oracle.go
│   └── geofences/
│       ├── geoFencesValidator.go
│       └── getGeofences.go
├── .env
├── go.mod
└── README.md
```

## Funcionalidades

- Conexión a base de datos Oracle
- Procesamiento de datos de geocercas
- Validación de puntos dentro de geocercas
- Actualización de eventos de geocercas 