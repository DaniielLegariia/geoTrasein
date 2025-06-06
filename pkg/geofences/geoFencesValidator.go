package geofences

import (
	"fmt"
	"log"
	"math"

	geo "github.com/paulmach/go.geo"
)

// Estructura que representa una geocerca
type Geofence struct {
	IDGeocerca   int
	Coordenadas  []*geo.Point // Coordenadas del polígono o puntos de la línea
	TipoGeocerca int          // 1: Circular, 2: Poligonal, 3: Lineal
	Radius       float64      // Si es una geocerca circular, esto es el radio
	Centro       *geo.Point   // Si es circular, este es el centro
	GeoEvento   int
}

// Estructura que representa el evento GPS
type Event struct {
	UnidadID string
	Lat      float64
	Lon      float64
}

// Clase principal para la creación y validación de geocercas
type GeofenceValidator struct {
	Geocercas []Geofence
}

// Constructor de la clase `GeofenceValidator`
func NewGeofenceValidator(idGeocerca int, tipoGeocerca int, coordenadas []*geo.Point, radio float64) *GeofenceValidator {
	var centro *geo.Point
	if tipoGeocerca == 1 { // Si es circular, necesitamos un centro
		centro = coordenadas[0] // Por simplicidad, la primera coordenada será el centro
	}

	geofence := Geofence{
		IDGeocerca:   idGeocerca,
		TipoGeocerca: tipoGeocerca,
		Coordenadas:  coordenadas,
		Radius:       radio,
		Centro:       centro,
	}

	return &GeofenceValidator{
		Geocercas: []Geofence{geofence},
	}
}

// Función para crear un evento GPS
func (g *GeofenceValidator) CreateEvent(unidadID string, lat, lon float64) Event {
	return Event{
		UnidadID: unidadID,
		Lat:      lat,
		Lon:      lon,
	}
}

// Función que recorre la lista de geocercas y valida según el tipo de geocerca
func (g *GeofenceValidator) Validate(event Event) int {
	for _, geofence := range g.Geocercas {
		switch geofence.TipoGeocerca {
		case 1:
			//log.Println("PARSER: Validando geocerca circular")
			// Validar geocerca circular
			if g.IsPointInCircularGeofence(geofence, event) {
				return 1 // Dentro de la geocerca circular
			}
		case 2:
			//log.Println("PARSER: Validando geocerca Poligonal")

			// Validar geocerca poligonal
			if g.IsPointInPolygonGeofence(geofence, event) {
				return 1 // Dentro de la geocerca poligonal
			}
		case 3:
			// Validar geocerca lineal
			//log.Println("PARSER: Validando geocerca lineal")
			if g.IsPointNearLineGeofence(geofence, event) {
				return 1 // Cerca de la geocerca lineal
			}
		default:
			fmt.Println("PARSER: Tipo de geocerca no soportado")
		}
	}
	return 2 // Fuera de todas las geocercas
}

// Validar si un punto está dentro de una geocerca circular
func (g *GeofenceValidator) IsPointInCircularGeofence(geofence Geofence, event Event) bool {
	if geofence.Centro == nil {
		log.Println("PARSER: Centro de la geocerca es nil")
		return false
	}
	//por algun motivo  lat y lon estan invertidos en el centro de la geocerca
	distance := HaversineDistance(geofence.Centro.Lng(), geofence.Centro.Lat(), event.Lat, event.Lon)
	return distance <= geofence.Radius
}

// Validar si un punto está dentro de una geocerca poligonal
func (g *GeofenceValidator) IsPointInPolygonGeofence(geofence Geofence, event Event) bool {
	point := geo.NewPoint(event.Lat, event.Lon)

	// Obtener las coordenadas del polígono en el formato correcto
	var polygonCoords [][]float64
	for _, coord := range geofence.Coordenadas {
		polygonCoords = append(polygonCoords, []float64{coord.Lng(), coord.Lat()})
	}

	// Usar el algoritmo de ray-casting para determinar si el punto está dentro del polígono
	return isPointInPolygon(point, polygonCoords)
}

// Validar si un punto está cerca de una línea (geocerca lineal)
func (g *GeofenceValidator) IsPointNearLineGeofence(geofence Geofence, event Event) bool {
	point := geo.NewPoint(event.Lat, event.Lon)

	for i := 0; i < len(geofence.Coordenadas)-1; i++ {
		start := geofence.Coordenadas[i]
		end := geofence.Coordenadas[i+1]

		distance := pointToLineDistance(point, start, end)
		if distance <= 50 { // Distancia de 50 metros
			return true
		}
	}
	return false
}

func pointToLineDistance(p, start, end *geo.Point) float64 {
	// Fórmula para la distancia entre un punto y una línea
	x0, y0 := p.X(), p.Y()
	x1, y1 := start.X(), start.Y()
	x2, y2 := end.X(), end.Y()

	num := math.Abs((y2-y1)*x0 - (x2-x1)*y0 + x2*y1 - y2*x1)
	den := math.Sqrt((y2-y1)*(y2-y1) + (x2-x1)*(x2-x1))

	return num / den
}

func isPointInPolygon(point *geo.Point, polygonCoords [][]float64) bool {
	x := point.X()
	y := point.Y()

	inside := false
	n := len(polygonCoords)
	for i := 0; i < n; i++ {
		x1 := polygonCoords[i][0]
		y1 := polygonCoords[i][1]
		x2 := polygonCoords[(i+1)%n][0]
		y2 := polygonCoords[(i+1)%n][1]

		if (y1 > y) != (y2 > y) && x < (x2-x1)*(y-y1)/(y2-y1)+x1 {
			inside = !inside
		}
	}
	return inside
}

// Función que calcula la distancia entre dos puntos utilizando la fórmula Haversine
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000 // Radio de la Tierra en metros
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c // Retorna la distancia en metros
} 