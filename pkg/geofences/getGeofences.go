package geofences

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"geoTrasein/pkg/data"

	"github.com/godror/godror"
	geo "github.com/paulmach/go.geo"
)

type GeocercasResult struct {
	Geocercas []Geofence
}

func ValidateGeofence(Imei string, lat, lon float64, DateGPS string, EventGPS string) {
	result, err := GetGeocercas(Imei)
	if err != nil {
		log.Print("PARSER Error al obtener geocercas:", err)
		return
	}

	if len(result.Geocercas) > 0 {
		for _, geocerca := range result.Geocercas {
			validator := NewGeofenceValidator(geocerca.IDGeocerca, geocerca.TipoGeocerca, geocerca.Coordenadas, geocerca.Radius)
			event := validator.CreateEvent(Imei, lat, lon)
			resultado := validator.Validate(event)
			if resultado == 1 && resultado != geocerca.GeoEvento {
				updateGeofenceEvent(geocerca.IDGeocerca, Imei, DateGPS, EventGPS, lat, lon, 1, 1)
			} else if resultado == 2 && resultado != geocerca.GeoEvento {
				updateGeofenceEvent(geocerca.IDGeocerca, Imei, DateGPS, EventGPS, lat, lon, 2, 1)
			}
		}
	} else {
		log.Printf("PARSER No se encontraron geocercas.")
	}
}

func GetGeocercas(Imei string) (GeocercasResult, error) {
	log.Print("PARSER ORACLE")
	defer func() {
		if err := recover(); err != nil {
			log.Println("cliente-PARSER-GEO CONEXION-back panic occurred:", err)
		}
	}()

	dbOra, err := data.GetEvetosPersistentOraDB()
	if err != nil {
		log.Print("PARSER-GEO: Error al obtener la conexión a la base de datos", err)
		return GeocercasResult{}, err
	}

	var rset driver.Rows

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_, err = dbOra.ExecContext(ctx, `begin DBRASTREO.prc_get_geocercas_parser(:1,:2); end;`, Imei, sql.Out{Dest: &rset})
	if err != nil {
		log.Print("PARSER-GEO: Error al ejecutar el SP prc_ws_get_equipos", err)
		return GeocercasResult{}, err
	}

	if rset == nil {
		return GeocercasResult{}, fmt.Errorf("equipo sin geocercas")
	}

	type rowType struct {
		IDGEOCERCA  int
		COORDENADAS string
		GEO_EVENTO  int
		TIPO_GEO    int
		RADIO       float64
		message     string
	}

	var geocercas []Geofence
	for {
		var row rowType
		rowI := make([]driver.Value, 6)

		if err := rset.Next(rowI); err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("PARSER-GEO: %+v", err)
			return GeocercasResult{}, err
		}

		processRow(rowI, &row)
		if row.COORDENADAS == "" {
			log.Printf("PARSER coordenadas inválidas o vacías para IDGeocerca: %d", row.IDGEOCERCA)
			continue
		}

		coordenadas, err := parseCoordinates(row.COORDENADAS)
		if err != nil {
			log.Printf("PARSER Error al parsear coordenadas: %v", err)
			return GeocercasResult{}, err
		}

		geocerca := Geofence{
			IDGeocerca:   row.IDGEOCERCA,
			Coordenadas:  coordenadas,
			TipoGeocerca: row.TIPO_GEO,
			Radius:       row.RADIO,
			GeoEvento:    row.GEO_EVENTO,
		}

		geocercas = append(geocercas, geocerca)
	}

	if len(geocercas) == 0 {
		return GeocercasResult{}, fmt.Errorf("PARSER equipo sin geocercas")
	}

	if rset != nil {
		defer rset.Close()
	}

	return GeocercasResult{
		Geocercas: geocercas,
	}, nil
}

func processRow(row []driver.Value, structType interface{}) {
	v := reflect.ValueOf(structType).Elem()
	for i, col := range row {
		field := v.Field(i)
		if !field.CanSet() {
			log.Printf("PARSER No se puede asignar al campo en índice %d", i)
			continue
		}

		switch v := col.(type) {
		case godror.Number:
			if field.Kind() == reflect.String {
				field.SetString(string(v))
			} else if field.Kind() == reflect.Int {
				if intValue, err := strconv.Atoi(string(v)); err == nil {
					field.SetInt(int64(intValue))
				} else {
					log.Printf("PARSER Error al convertir godror.Number a int: %v", err)
				}
			} else if field.Kind() == reflect.Float64 {
				if floatValue, err := strconv.ParseFloat(string(v), 64); err == nil {
					field.SetFloat(floatValue)
				} else {
					log.Printf("PARSER Error al convertir godror.Number a float64: %v", err)
				}
			}
		case string:
			if field.Kind() == reflect.String {
				field.SetString(v)
			}
		case sql.NullString:
			if field.Kind() == reflect.String {
				if v.Valid {
					field.SetString(v.String)
				} else {
					field.SetString("")
				}
			}
		case sql.NullInt64:
			if field.Kind() == reflect.Int {
				if v.Valid {
					field.SetInt(v.Int64)
				} else {
					field.SetInt(0)
				}
			}
		case sql.NullFloat64:
			if field.Kind() == reflect.Float64 {
				if v.Valid {
					field.SetFloat(v.Float64)
				} else {
					field.SetFloat(0.0)
				}
			}
		case int64:
			if field.Kind() == reflect.Int {
				field.SetInt(v)
			}
		case float64:
			if field.Kind() == reflect.Float64 {
				field.SetFloat(v)
			}
		default:
			log.Printf("PARSER Tipo de dato no manejado: %T", v)
		}
	}
}

func parseCoordinates(coordinates string) ([]*geo.Point, error) {
	var points []*geo.Point
	coordinates = strings.TrimSuffix(coordinates, "|")
	coords := strings.Split(coordinates, "|")
	if len(coords)%2 != 0 {
		return nil, fmt.Errorf("PARSER formato de coordenadas inválido: %s", coordinates)
	}
	for i := 0; i < len(coords); i += 2 {
		lat, err1 := strconv.ParseFloat(coords[i], 64)
		lon, err2 := strconv.ParseFloat(coords[i+1], 64)
		if err1 == nil && err2 == nil {
			points = append(points, geo.NewPoint(lat, lon))
		} else {
			return nil, fmt.Errorf("PARSER error al parsear coordenadas: %v, %v", err1, err2)
		}
	}
	return points, nil
}

func updateGeofenceEvent(idGEo int, Imei string, DateGPs string, Evengps string, lat float64, lon float64, GeoEvento int, messageType int) error {
	log.Print("PARSER Entra a updateGeofenceEvent: ", idGEo, Imei, DateGPs, Evengps, lat, lon, GeoEvento, messageType)
	dbOra, err := data.GetEvetosPersistentOraDB()
	if err != nil {
		log.Print("PARSER-GEO: Error al obtener la conexión a la base de datos", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Función para formatear la fecha en el formato que Oracle espera
	dateGPS := formatDateForOracle(DateGPs)

	_, err = dbOra.ExecContext(ctx, `begin DBRASTREO.prc_actualiza_geopoligonal(:1,:2,:3,:4,:5,:6,:7,:8); end;`, idGEo, Imei, dateGPS, Evengps, lat, lon, GeoEvento, messageType)
	if err != nil {
		log.Print("PARSER-GEO: Error al ejecutar el SP prc_actualiza_geopoligonal", err)
		return err
	}

	return nil
}

// Función para formatear la fecha en el formato que Oracle espera
func formatDateForOracle(dateStr string) string {
	// Primero intentamos parsear la fecha
	t, err := time.Parse("2006-01-02 15:04:05", dateStr)
	if err != nil {
		// Si hay error, intentamos con otro formato común
		t, err = time.Parse(time.RFC3339, dateStr)
		if err != nil {
			log.Printf("Error al parsear la fecha %s: %v", dateStr, err)
			return dateStr
		}
	}
	// Formateamos la fecha en el formato que Oracle espera: DD-MON-YYYY HH24:MI:SS
	return t.Format("02-Jan-2006 15:04:05")
}
