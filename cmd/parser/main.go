package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"geoTrasein/pkg/data"
	"geoTrasein/pkg/geofences"

	"github.com/godror/godror"
	"github.com/joho/godotenv"
)

type EventoGeo struct {
	Imei     string
	Lat      float64
	Lon      float64
	DateGPS  string
	EventGPS string
}

func main() {
	log.Println("Iniciando aplicación...")

	// Cargar variables de entorno desde .env
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error al cargar el archivo .env")
	}
	log.Println("Variables de entorno cargadas correctamente")

	// Configurar el logger
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Logger configurado")

	// Obtener conexión a Oracle
	log.Println("Solicitando conexión a Oracle...")
	dbOra, err := data.GetEvetosPersistentOraDB()
	if err != nil {
		log.Fatalf("Error al conectar a Oracle: %v", err)
	}
	defer dbOra.Close()
	log.Println("Conexión a Oracle establecida correctamente")

	// Crear contexto con timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	log.Println("Contexto creado con timeout de 5 minutos")

	// Ejecutar el procedimiento almacenado
	log.Println("Ejecutando procedimiento almacenado...")
	var rset driver.Rows
	_, err = dbOra.ExecContext(ctx, `begin DBRASTREO.prc_ws_nestle_get_evento_geo_trasein(:1); end;`, sql.Out{Dest: &rset})
	if err != nil {
		log.Fatalf("Error al ejecutar el SP: %v", err)
	}
	defer rset.Close()
	log.Println("Procedimiento almacenado ejecutado correctamente")

	// Procesar los resultados
	log.Println("Iniciando procesamiento de resultados...")
	contador := 0
	for {
		row := make([]driver.Value, 5)
		err := rset.Next(row)
		if err != nil {
			if err == io.EOF {
				log.Println("Fin de los resultados")
				break
			}
			log.Printf("Error al leer la siguiente fila: %v", err)
			continue
		}

		// Convertir los valores a la estructura EventoGeo
		lat, _ := strconv.ParseFloat(row[1].(godror.Number).String(), 64)
		lon, _ := strconv.ParseFloat(row[2].(godror.Number).String(), 64)

		// Convertir DateGPS a string (puede ser time.Time o string)
		var dateGPS string
		switch v := row[3].(type) {
		case time.Time:
			dateGPS = v.Format("2006-01-02 15:04:05")
		case string:
			dateGPS = v
		default:
			dateGPS = fmt.Sprintf("%v", v)
		}

		// Convertir IMEI a string (puede ser int64 o string)
		var imei string
		switch v := row[0].(type) {
		case string:
			imei = v
		case int64:
			imei = strconv.FormatInt(v, 10)
		default:
			imei = fmt.Sprintf("%v", v)
		}

		// Convertir EventGPS a string (puede ser int64 o string)
		var eventGPS string
		switch v := row[4].(type) {
		case string:
			eventGPS = v
		case int64:
			eventGPS = strconv.FormatInt(v, 10)
		default:
			eventGPS = fmt.Sprintf("%v", v)
		}

		evento := EventoGeo{
			Imei:     imei,
			Lat:      lat,
			Lon:      lon,
			DateGPS:  dateGPS,
			EventGPS: eventGPS,
		}

		// Log del primer registro para verificar
		if contador == 0 {
			log.Printf("Primer registro procesado: %+v", evento)
		}

		// Procesar el evento
		geofences.ValidateGeofence(evento.Imei, evento.Lat, evento.Lon, evento.DateGPS, evento.EventGPS)

		contador++
		if contador%1000 == 0 {
			log.Printf("Procesados %d registros", contador)
		}
	}

	log.Printf("Proceso completado. Total de registros procesados: %d", contador)
}
