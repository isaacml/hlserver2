package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"net/http"
	"os"
	"strings"
	"time"
)

type Grafico struct {
	Type    string   `json:"type"`
	Data    []int    `json:"data"`
	Labels  []string `json:"labels"`
	Colores []string `json:"colores"`
}

func giveFecha(w http.ResponseWriter, r *http.Request) {
	anio, mes, dia := time.Now().Date()                    //Fecha actual
	fecha := fmt.Sprintf("%02d/%02d/%02d", dia, mes, anio) // Fecha Actual
	fmt.Fprintf(w, "fecha=%s", fecha)
}

func zeroFields(w http.ResponseWriter, r *http.Request) {
	var existe int
	anio, mes, dia := time.Now().Date()                           //Fecha actual
	fecha_actual := fmt.Sprintf("%02d-%02d-%02d", anio, mes, dia) // Fecha actual
	// La primera vez que se entra a la web, se abre el fichero de dayly.db actual
	db0, err := sql.Open("sqlite3", dirDaylys+fecha_actual+"dayly.db")
	if err != nil {
		Error.Println(err)
	}
	defer db0.Close()
	dbday_mu.RLock()
	row := db0.QueryRow("SELECT count(*) FROM resumen WHERE username = ?", username).Scan(&existe)
	dbday_mu.RUnlock()
	if row != nil {
		Warning.Println(row)
	}
	if existe == 0 {
		fmt.Fprintf(w, "Nada")
	} else {
		fmt.Fprintf(w, "Hay")
	}
}

func firstFecha(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var (
		arrSo, arrIso, paisSes                          []string
		arrTime, arrSess, timePais, sesionPais, sesHour []int
	)
	var horaSes map[int]int = make(map[int]int)
	anio, mes, dia := time.Now().Date()                   //Fecha actual
	colores := []string{"#F9183A", "#F918E6", "#4118F9", "#18DBF9", "#18F9D3", "#18F950", "#C4F918", "#EEF918", "#F9C118", "#0E0B01"}  //Colores para graficos1 Paises
	colores2 := []string{"#FFCE56", "#36A2EB", "#FF6384", "#00ff17" } //Colores para graficos2 OS
	//Fecha actual
	fecha_actual := fmt.Sprintf("%02d-%02d-%02d", anio, mes, dia) // Fecha actual
	fecha_ESP := fmt.Sprintf("%02d/%02d/%02d", dia, mes, anio)    // Fecha a mostrar en el html
	fecha_ESP = "Estadísticas correspondientes al día " + fecha_ESP
	// La primera vez que se entra a la web, se abre el fichero de dayly.db actual
	db_now, err := sql.Open("sqlite3", dirDaylys+fecha_actual+"dayly.db")
	if err != nil {
		Error.Println(err)
	}
	dbday_mu.RLock()
	query, err := db_now.Query("SELECT time, os, count FROM resumen WHERE username = ? GROUP BY username, streamname, os", username)
	dbday_mu.RUnlock()
	if err != nil {
		Warning.Println(err)
	}
	for query.Next() {
		var time, count int
		var so string
		err = query.Scan(&time, &so, &count)
		if err != nil {
			Warning.Println(err)
		}
		arrTime = append(arrTime, time)
		arrSo = append(arrSo, so)
		arrSess = append(arrSess, count)
	}
	dbday_mu.RLock()
	query2, err := db_now.Query("SELECT sum(time), isocode FROM resumen WHERE username = ? AND time IN (SELECT time FROM resumen GROUP BY username, streamname, isocode, os) GROUP BY isocode", username)
	dbday_mu.RUnlock()
	if err != nil {
		Error.Println(err)
	}
	for query2.Next() {
		var time int
		var isocode string
		err = query2.Scan(&time, &isocode)
		if err != nil {
			Warning.Println(err)
		}
		timePais = append(timePais, time)
		arrIso = append(arrIso, isocode)
	}
	dbday_mu.RLock()
	query3, err := db_now.Query("SELECT sum(count), isocode FROM resumen WHERE username = ? AND id IN(SELECT id FROM resumen GROUP BY username, streamname, isocode , os HAVING count = max(count))  GROUP BY isocode ", username)
	dbday_mu.RUnlock()
	if err != nil {
		Error.Println(err)
	}
	for query3.Next() {
		var count int
		var isocode string
		err = query3.Scan(&count, &isocode)
		if err != nil {
			Warning.Println(err)
		}
		sesionPais = append(sesionPais, count)
		paisSes = append(paisSes, isocode)
	}
	dbday_mu.RLock()
	query4, err := db_now.Query("SELECT sum(count), hour FROM resumen WHERE username = ? AND id IN(SELECT id FROM resumen GROUP BY username, streamname, isocode, hour, os HAVING count = max(count))  GROUP BY hour ORDER BY hour ASC", username)
	dbday_mu.RUnlock()
	if err != nil {
		Error.Println(err)
	}
	for query4.Next() {
		var count, hora int
		err = query4.Scan(&count, &hora)
		if err != nil {
			Warning.Println(err)
		}
		sesHour = onlyHours()
		horaSes[hora] = count
	}
	// Aquí se crean los JSON
	grafico0, _ := json.Marshal(Grafico{"pie", arrTime, arrSo, colores2})        // Aquí se crea el JSON para el grafico de segundos consumidos por sistema operativo
	grafico1, _ := json.Marshal(Grafico{"pie", arrSess, arrSo, colores2})        // Aquí se crea el JSON para el grafico de sesiones por sistema operativo
	grafico2, _ := json.Marshal(Grafico{"pie", timePais, arrIso, colores})       // Aquí se crea el JSON para el grafico de segundos consumidos por pais
	grafico3, _ := json.Marshal(Grafico{"pie", sesionPais, paisSes, colores})    // Aquí se crea el JSON para el grafico de sesiones por pais
	grafico4, _ := json.Marshal(Grafico2{"line", sesionHours(horaSes), sesHour}) // Aquí se crea el JSON para el grafico de sesiones por franja horaria
	fmt.Fprintf(w, "%s;%s;%s;%s;%s;%s", fecha_ESP, string(grafico0), string(grafico1), string(grafico2), string(grafico3), string(grafico4))
	db_now.Close()
}

func formatDaylyhtml(w http.ResponseWriter, r *http.Request) {
	canv1 := "<h3>Sesiones por Franja Horaria</h3><canvas id='selGraf5'/>"
	title := "<h3>Plataformas Usadas</h3>"
	canv2 := "<label>Sesiones por Sistema Operativo</label><canvas id='selGraf2'/>"
	canv3 := "<label>Segundos consumidos por Sistema Operativo</label><canvas id='selGraf1'/>"
	title2 := "<h3>Datos de paises</h3>"
	canv4 := "<label>Tiempo consumido(en Sec) por País</label><canvas id='selGraf3'/>"
	canv5 := "<label>Sesiones por País</label><canvas id='selGraf4'/>"

	fmt.Fprintf(w, "%s;%s;%s;%s;%s;%s;%s", canv1, title, canv2, canv3, title2, canv4, canv5)
}

func consultaFecha(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() // recupera campos del form tanto GET como POST
	var (
		arrSo, arrIso, paisSes                          []string
		arrTime, arrSess, timePais, sesionPais, sesHour []int
	)
	var horaSes map[int]int = make(map[int]int)
	colores := []string{"#F9183A", "#F918E6", "#4118F9", "#18DBF9", "#18F9D3", "#18F950", "#C4F918", "#EEF918", "#F9C118", "#0E0B01"}  //Colores para graficos1 Paises
	colores2 := []string{"#FFCE56", "#36A2EB", "#FF6384", "#00ff17" } //Colores para graficos2 OS
	//Fecha obtenida del select de dayly.html
	fechaHTML := strings.Split(r.FormValue("fecha"), "/")
	fechaSQL := fmt.Sprintf("%s-%s-%s", fechaHTML[2], fechaHTML[1], fechaHTML[0]) // Formato SQLite
	fechaESP := "Estadísticas correspondientes al día " + r.FormValue("fecha")    // Fecha a mostrar en HTML
	//Al escoger una fecha, comprobamos si existe el fichero de Base de datos
	if _, err := os.Stat(dirDaylys + fechaSQL + "dayly.db"); os.IsNotExist(err) {
		Warning.Println("No existe el fichero de base de datos.")
		fmt.Fprintf(w, "NoBD")
	} else {
		//Por lo tanto, abrimos el fichero
		db_fecha, err := sql.Open("sqlite3", dirDaylys+fechaSQL+"dayly.db")
		if err != nil {
			Warning.Println(err)
		}
		dbday_mu.RLock()
		exist, err := db_fecha.Query("SELECT * FROM resumen WHERE username = ?", username)
		dbday_mu.RUnlock()
		if err != nil {
			Warning.Println(err)
		}
		if exist.Next() == false {
			Warning.Println("Fichero de base de datos vacío.")
			fmt.Fprintf(w, "NoBD")
		} else {
			dbday_mu.RLock()
			query, err := db_fecha.Query("SELECT time, os, count FROM resumen WHERE username = ? GROUP BY username, streamname, os", username)
			dbday_mu.RUnlock()
			if err != nil {
				Warning.Println(err)
			}
			for query.Next() {
				var time, count int
				var so string
				err = query.Scan(&time, &so, &count)
				if err != nil {
					Warning.Println(err)
				}
				arrTime = append(arrTime, time)
				arrSo = append(arrSo, so)
				arrSess = append(arrSess, count)
			}
			dbday_mu.RLock()
			query2, err := db_fecha.Query("SELECT sum(time), isocode FROM resumen WHERE username = ? AND time IN (SELECT time FROM resumen GROUP BY username, streamname, isocode, os) GROUP BY isocode", username)
			dbday_mu.RUnlock()
			if err != nil {
				Error.Println(err)
			}
			for query2.Next() {
				var time int
				var isocode string
				err = query2.Scan(&time, &isocode)
				if err != nil {
					Warning.Println(err)
				}
				timePais = append(timePais, time)
				arrIso = append(arrIso, isocode)
			}
			dbday_mu.RLock()
			query3, err := db_fecha.Query("SELECT sum(count), isocode FROM resumen WHERE username = ? AND id IN(SELECT id FROM resumen GROUP BY username, streamname, isocode , os HAVING count = max(count))  GROUP BY isocode", username)
			dbday_mu.RUnlock()
			if err != nil {
				Error.Println(err)
			}
			for query3.Next() {
				var count int
				var isocode string
				err = query3.Scan(&count, &isocode)
				if err != nil {
					Warning.Println(err)
				}
				sesionPais = append(sesionPais, count)
				paisSes = append(paisSes, isocode)
			}
			dbday_mu.RLock()
			query4, err := db_fecha.Query("SELECT sum(count), hour FROM resumen WHERE username = ? AND id IN(SELECT id FROM resumen GROUP BY username, streamname, isocode, hour, os HAVING count = max(count))  GROUP BY hour ORDER BY hour ASC", username)
			dbday_mu.RUnlock()
			if err != nil {
				Error.Println(err)
			}
			for query4.Next() {
				var count int
				var hora int
				err = query4.Scan(&count, &hora)
				if err != nil {
					Warning.Println(err)
				}
				sesHour = onlyHours()
				horaSes[hora] = count
			}
			// Aquí se crean los JSON
			grafico0, _ := json.Marshal(Grafico{"pie", arrTime, arrSo, colores2})        // Aquí se crea el JSON para el grafico de segundos consumidos por sistema operativo
			grafico1, _ := json.Marshal(Grafico{"pie", arrSess, arrSo, colores2})        // Aquí se crea el JSON para el grafico de sesiones por sistema operativo
			grafico2, _ := json.Marshal(Grafico{"pie", timePais, arrIso, colores})       // Aquí se crea el JSON para el grafico de segundos consumidos por pais
			grafico3, _ := json.Marshal(Grafico{"pie", sesionPais, paisSes, colores})    // Aquí se crea el JSON para el grafico de sesiones por pais
			grafico4, _ := json.Marshal(Grafico2{"line", sesionHours(horaSes), sesHour}) // Aquí se crea el JSON para el grafico de sesiones por franja horaria
			fmt.Fprintf(w, "%s;%s;%s;%s;%s;%s", fechaESP, string(grafico0), string(grafico1), string(grafico2), string(grafico3), string(grafico4))
		}
		db_fecha.Close()
	}
}

//funcion que va a generar la horas de un día
func onlyHours() []int {
	var sesHour []int
	for i := 1; i <= 24; i++ {
		sesHour = append(sesHour, i)
	}
	return sesHour
}

//funcion que va a colocar las sessiones en sus correspondientes horas
func sesionHours(hora map[int]int) []int {
	x := make([]int, 24)
	for cont, _ := range x {
		for key, value := range hora {
			if key == cont+1 {
				x[cont] = value
			}
		}
	}
	return x
}
