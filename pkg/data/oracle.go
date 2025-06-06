package data

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/godror/godror"
)

const oraConnStringsEvnts = `user="RASWEB" password="dbwebtcv" connectString="54.161.252.40:1530/dbrastcv?connect_timeout=20" poolSessionTimeout=20s poolSessionMaxLifetime=20s poolWaitTimeout=20s poolMinSessions=0 poolMaxSessions=50`

var oraDBEvents *sql.DB

func GetEvetosPersistentOraDB() (*sql.DB, error) {
	log.Println("Iniciando conexión a Oracle...")

	// Si la conexión persistente ya está establecida, la reutilizamos
	if oraDBEvents != nil {
		log.Println("Reutilizando conexión existente a Oracle")
		return oraDBEvents, nil
	}

	// Establecer la conexión persistente
	log.Println("Estableciendo nueva conexión a Oracle...")
	db, err := sql.Open("godror", oraConnStringsEvnts)
	if err != nil {
		log.Printf("Error al abrir la conexión: %v", err)
		return nil, err
	}

	// Verificar la conexión
	log.Println("Verificando conexión con ping...")
	if err := db.Ping(); err != nil {
		log.Printf("Error al hacer ping a la base de datos: %v", err)
		return nil, err
	}
	log.Println("Conexión exitosa a Oracle")

	// Ajustar la configuración de la conexión
	log.Println("Configurando pool de conexiones...")
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(100)
	db.SetConnMaxLifetime(5 * time.Minute)

	oraDBEvents = db // Guardar la conexión persistente para reutilización futura
	log.Println("Conexión persistente establecida y configurada")

	return db, nil
}
