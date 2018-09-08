package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Grafico2 struct {
	Type   string `json:"type"`
	Data   []int  `json:"data"`
	Labels []int  `json:"labels"`
}
type Grafico3 struct {
	Type   string    `json:"type"`
	Data   []float64 `json:"data"`
	Labels []int     `json:"labels"`
}

func getMonthsYears(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var menu1, menu2, menu3 string
	anio, _, _ := time.Now().Date() //Fecha actual
	//Generamos el select de meses
	meses := []string{"Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio", "Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre"}
	_, mm, _ := time.Now().Date()
	for key, value := range meses {

		if int(mm) == key+1 {
			menu1 += fmt.Sprintf("<option label='%s' value='%02d' selected>%s</option>", value, key+1, value)
		} else {
			menu1 += fmt.Sprintf("<option label='%s' value='%02d'>%s</option>", value, key+1, value)
		}
	}
	//Generamos el select de años
	for _, value := range UpDownYears(anio) {
		if int(anio) == value {
			menu2 += fmt.Sprintf("<option label='%d' value='%d' selected>%d</option>", value, value, value)
		} else {
			menu2 += fmt.Sprintf("<option label='%d' value='%d'>%d</option>", value, value, value)
		}
	}
	menu3 = "<option label='todo' value='todo'>Todo</option>"
	fmt.Fprintf(w, "%s;%s;%s", menu1, menu2, menu3)
}

func firstMonthly(w http.ResponseWriter, r *http.Request) {
	cookie, err3 := r.Cookie(CookieName)
	if err3 != nil {
		return
	}
	key := cookie.Value
	mu_user.Lock()
	usr, ok := user[key] // De aquí podemos recoger el usuario
	mu_user.Unlock()
	if !ok {
		return
	}
	username := usr
	var fechaAud []int
	var menu3 string
	var arrAud map[int]int = make(map[int]int)
	var arrMin map[int]int = make(map[int]int)
	var arrAVG map[int]float64 = make(map[int]float64)
	var arrMegas map[int]int = make(map[int]int)
	var arrPico map[int]int = make(map[int]int)
	var horaPico map[int]int = make(map[int]int)
	anio, mes, _ := time.Now().Date() //Fecha actual
	mesGrafico := fmt.Sprintf("%d-%02d", anio, mes)
	db0, err := sql.Open("sqlite3", dirMonthlys+mesGrafico+"monthly.db")
	if err != nil {
		Error.Println(err)
	}
	//Generamos el select de streams
	dbmon_mu.RLock()
	query2, err := db0.Query("SELECT  DISTINCT(streamname) FROM resumen WHERE username = ?", username)
	dbmon_mu.RUnlock()
	if err != nil {
		Error.Println(err)
		return
	}
	if !query2.Next() {
		//No, campos vacios
		menu3 = "<option label='todo' value='todo' selected>Todo</option>"
	} else {
		//Si, existen campos. Formamos el select
		menu3 = "<option label='todo' value='todo' selected>Todo</option>"
		dbmon_mu.RLock()
		query3, err := db0.Query("SELECT  DISTINCT(streamname) FROM resumen WHERE username = ?", username)
		dbmon_mu.RUnlock()
		if err != nil {
			Error.Println(err)
			return
		}
		for query3.Next() {
			var stream string
			err = query3.Scan(&stream)
			if err != nil {
				Warning.Println(err)
				continue
			}
			menu3 += "<option label='" + stream + "' value='" + stream + "'>" + stream + "</option>"
		}
		query3.Close()
	}
	query2.Close()
	// Se añaden los dia del mes al grafico
	for i := 1; i <= daysIn(mes, anio); i++ {
		fechaAud = append(fechaAud, i)
	}
	dbmon_mu.RLock()
	query, err := db0.Query("SELECT sum(audiencia), sum(minutos), avg(minutos), sum(megabytes), max(pico), horapico, fecha FROM resumen WHERE username = ? GROUP BY fecha", username)
	dbmon_mu.RUnlock()
	if err != nil {
		Error.Println(err)
		return
	}
	for query.Next() {
		var audiencia, minutos, megas, pico int
		var promedio float64
		var fecha, horapico string
		err = query.Scan(&audiencia, &minutos, &promedio, &megas, &pico, &horapico, &fecha)
		if err != nil {
			Warning.Println(err)
			continue
		}
		hour := strings.Split(horapico, ":")
		day := strings.Split(fecha, ":")
		arrAud[toInt(day[1])] = audiencia
		arrMin[toInt(day[1])] = minutos
		arrAVG[toInt(day[1])] = promedio
		arrMegas[toInt(day[1])] = megas
		arrPico[toInt(day[1])] = pico
		horaPico[toInt(day[1])] = toInt(hour[0])
	}
	query.Close()
	g1 := grafDays(arrAud, len(fechaAud))
	g2 := grafDays(arrMin, len(fechaAud))
	g3 := grafDaysFloat(arrAVG, len(fechaAud))
	g4 := grafDays(arrMegas, len(fechaAud))
	g5 := grafDays(arrPico, len(fechaAud))
	g6 := grafDays(horaPico, len(fechaAud))
	// Aquí se crean los JSON
	grafico1, _ := json.Marshal(Grafico2{"bar", g1, fechaAud})  // Aquí se crea el JSON para el grafico de audiencia total del dia en personas
	grafico2, _ := json.Marshal(Grafico2{"bar", g2, fechaAud})  // Aquí se crea el JSON para el grafico de tiempo total visionado
	grafico3, _ := json.Marshal(Grafico3{"bar", g3, fechaAud})  // Aquí se crea el JSON para el grafico de segundos consumidos por pais
	grafico4, _ := json.Marshal(Grafico2{"bar", g4, fechaAud})  // Aquí se crea el JSON para el grafico de sesiones por pais
	grafico5, _ := json.Marshal(Grafico2{"bar", g5, fechaAud})  // Aquí se crea el JSON para el grafico de sesiones por franja horaria
	grafico6, _ := json.Marshal(Grafico2{"line", g6, fechaAud}) // Aquí se crea el JSON para el grafico de sesiones por franja horaria
	fmt.Fprintf(w, "%s;%s;%s;%s;%s;%s;%s", string(grafico1), string(grafico2), string(grafico3), string(grafico4), string(grafico5), string(grafico6), menu3)
	db0.Close()
}

func graficosMonthly(w http.ResponseWriter, r *http.Request) {
	cookie, err3 := r.Cookie(CookieName)
	if err3 != nil {
		return
	}
	key := cookie.Value
	mu_user.Lock()
	usr, ok := user[key] // De aquí podemos recoger el usuario
	mu_user.Unlock()
	if !ok {
		return
	}
	username := usr
	r.ParseForm() // recupera campos del form tanto GET como POST
	var (
		arrNull, fechaAud []int
	)
	var arrAud map[int]int = make(map[int]int)
	var arrMin map[int]int = make(map[int]int)
	var arrAVG map[int]float64 = make(map[int]float64)
	var arrMegas map[int]int = make(map[int]int)
	var arrPico map[int]int = make(map[int]int)
	var horaPico map[int]int = make(map[int]int)
	mesGrafico := r.FormValue("years") + "-" + r.FormValue("months")
	// Se añaden los dias del mes al grafico
	for i := 1; i <= daysStringIn(r.FormValue("months"), toInt(r.FormValue("years"))); i++ {
		fechaAud = append(fechaAud, i)
	}
	//Se comprueba si existe la base de datos mensual
	if _, err := os.Stat(dirMonthlys + mesGrafico + "monthly.db"); os.IsNotExist(err) {
		//No hay base de datos
		Warning.Println("No existe el fichero de base de datos")
		Error.Println(err)
		tipo1, _ := json.Marshal(Grafico2{"bar", arrNull, fechaAud})
		menu3 := "<option label='todo' value='todo'>Todo</option>"
		fmt.Fprintf(w, "%s;%s;%s;%s;%s;%s;%s", string(tipo1), string(tipo1), string(tipo1), string(tipo1), string(tipo1), string(tipo1), menu3)
	} else {
		if r.FormValue("stream") == "todo" {
			var menu3 string
			db0, err := sql.Open("sqlite3", dirMonthlys+mesGrafico+"monthly.db")
			if err != nil {
				Error.Println(err)
			}
			//Generamos el select de streams
			dbmon_mu.RLock()
			query2, err := db0.Query("SELECT  DISTINCT(streamname) FROM resumen WHERE username = ?", username)
			dbmon_mu.RUnlock()
			if err != nil {
				Error.Println(err)
				return
			}
			if !query2.Next() {
				//No, campos vacios
				menu3 = "<option label='todo' value='todo' selected>Todo</option>"
			} else {
				//Si, existen campos. Formamos el select
				menu3 = "<option label='todo' value='todo' selected>Todo</option>"
				dbmon_mu.RLock()
				query3, err := db0.Query("SELECT  DISTINCT(streamname) FROM resumen WHERE username = ?", username)
				dbmon_mu.RUnlock()
				if err != nil {
					Warning.Println(err)
					return
				}
				for query3.Next() {
					var stream string
					err = query3.Scan(&stream)
					if err != nil {
						Warning.Println(err)
						continue
					}
					menu3 += "<option label='" + stream + "' value='" + stream + "'>" + stream + "</option>"
				}
				query3.Close()
			}
			query2.Close()
			dbmon_mu.RLock()
			query, err := db0.Query("SELECT sum(audiencia), sum(minutos), avg(minutos), sum(megabytes), max(pico), horapico, fecha FROM resumen WHERE username = ? GROUP BY fecha", username)
			dbmon_mu.RUnlock()
			if err != nil {
				Error.Println(err)
				return
			}
			for query.Next() {
				var audiencia, minutos, megas, pico int
				var promedio float64
				var horapico, fecha string
				err = query.Scan(&audiencia, &minutos, &promedio, &megas, &pico, &horapico, &fecha)
				if err != nil {
					Warning.Println(err)
					continue
				}
				hour := strings.Split(horapico, ":")
				day := strings.Split(fecha, ":")
				arrAud[toInt(day[1])] = audiencia
				arrMin[toInt(day[1])] = minutos
				arrAVG[toInt(day[1])] = promedio
				arrMegas[toInt(day[1])] = megas
				arrPico[toInt(day[1])] = pico
				horaPico[toInt(day[1])] = toInt(hour[0])
			}
			query.Close()
			//Se seneran los gŕaficos
			g1 := grafDays(arrAud, len(fechaAud))
			g2 := grafDays(arrMin, len(fechaAud))
			g3 := grafDaysFloat(arrAVG, len(fechaAud))
			g4 := grafDays(arrMegas, len(fechaAud))
			g5 := grafDays(arrPico, len(fechaAud))
			g6 := grafDays(horaPico, len(fechaAud))
			// Aquí se crean los JSON
			grafico0, _ := json.Marshal(Grafico2{"bar", g1, fechaAud})  // Aquí se crea el JSON para el grafico de audiencia total del dia en personas
			grafico1, _ := json.Marshal(Grafico2{"bar", g2, fechaAud})  // Aquí se crea el JSON para el grafico de tiempo total visionado
			grafico2, _ := json.Marshal(Grafico3{"bar", g3, fechaAud})  // Aquí se crea el JSON para el grafico de segundos consumidos por pais
			grafico3, _ := json.Marshal(Grafico2{"bar", g4, fechaAud})  // Aquí se crea el JSON para el grafico de sesiones por pais
			grafico4, _ := json.Marshal(Grafico2{"bar", g5, fechaAud})  // Aquí se crea el JSON para el grafico de sesiones por franja horaria
			grafico5, _ := json.Marshal(Grafico2{"line", g6, fechaAud}) // Aquí se crea el JSON para el grafico de sesiones por franja horaria
			fmt.Fprintf(w, "%s;%s;%s;%s;%s;%s;%s", string(grafico0), string(grafico1), string(grafico2), string(grafico3), string(grafico4), string(grafico5), menu3)
			db0.Close()
		} else {
			db0, err := sql.Open("sqlite3", dirMonthlys+mesGrafico+"monthly.db")
			if err != nil {
				Error.Println(err)
			}
			dbmon_mu.RLock()
			query2, err := db0.Query("SELECT  DISTINCT(streamname) FROM resumen WHERE username = ?", username)
			dbmon_mu.RUnlock()
			if err != nil {
				Error.Println(err)
				return
			}
			//Generamos el select de streams
			menu3 := "<option label='todo' value='todo'>Todo</option>"
			//Si existen campos, formamos el select
			for query2.Next() {
				var stream string
				err = query2.Scan(&stream)
				if err != nil {
					Warning.Println(err)
					continue
				}
				if stream == r.FormValue("stream") {
					menu3 += "<option label='" + stream + "' value='" + stream + "' selected>" + stream + "</option>"
				} else {
					menu3 += "<option label='" + stream + "' value='" + stream + "'>" + stream + "</option>"
				}
			}
			query2.Close()
			dbmon_mu.RLock()
			query, err := db0.Query("SELECT sum(audiencia), sum(minutos), avg(minutos), sum(megabytes), max(pico), horapico, fecha FROM resumen WHERE username = ? AND streamname = ? GROUP BY fecha", username, r.FormValue("stream"))
			dbmon_mu.RUnlock()
			if err != nil {
				Error.Println(err)
				return
			}
			for query.Next() {
				var audiencia, minutos, megas, pico int
				var promedio float64
				var horapico, fecha string
				err = query.Scan(&audiencia, &minutos, &promedio, &megas, &pico, &horapico, &fecha)
				if err != nil {
					Warning.Println(err)
					continue
				}
				hour := strings.Split(horapico, ":")
				day := strings.Split(fecha, ":")
				arrAud[toInt(day[1])] = audiencia
				arrMin[toInt(day[1])] = minutos
				arrAVG[toInt(day[1])] = promedio
				arrMegas[toInt(day[1])] = megas
				arrPico[toInt(day[1])] = pico
				horaPico[toInt(day[1])] = toInt(hour[0])
			}
			query.Close()
			//Se seneran los gŕaficos
			g1 := grafDays(arrAud, len(fechaAud))
			g2 := grafDays(arrMin, len(fechaAud))
			g3 := grafDaysFloat(arrAVG, len(fechaAud))
			g4 := grafDays(arrMegas, len(fechaAud))
			g5 := grafDays(arrPico, len(fechaAud))
			g6 := grafDays(horaPico, len(fechaAud))
			// Se crean los JSON
			grafico0, _ := json.Marshal(Grafico2{"bar", g1, fechaAud})  // Aquí se crea el JSON para el grafico de audiencia total del dia en personas
			grafico1, _ := json.Marshal(Grafico2{"bar", g2, fechaAud})  // Aquí se crea el JSON para el grafico de tiempo total visionado
			grafico2, _ := json.Marshal(Grafico3{"bar", g3, fechaAud})  // Aquí se crea el JSON para el grafico de segundos consumidos por pais
			grafico3, _ := json.Marshal(Grafico2{"bar", g4, fechaAud})  // Aquí se crea el JSON para el grafico de sesiones por pais
			grafico4, _ := json.Marshal(Grafico2{"bar", g5, fechaAud})  // Aquí se crea el JSON para el grafico de sesiones por franja horaria
			grafico5, _ := json.Marshal(Grafico2{"line", g6, fechaAud}) // Aquí se crea el JSON para el grafico de sesiones por franja horaria
			fmt.Fprintf(w, "%s;%s;%s;%s;%s;%s;%s", string(grafico0), string(grafico1), string(grafico2), string(grafico3), string(grafico4), string(grafico5), menu3)
			db0.Close()
		}
	}
}

// Funcion que muestra el total de horas y gigas consumidos (por primera vez)
func totalMonths(w http.ResponseWriter, r *http.Request) {
	cookie, err3 := r.Cookie(CookieName)
	if err3 != nil {
		return
	}
	key := cookie.Value
	mu_user.Lock()
	usr, ok := user[key] // De aquí podemos recoger el usuario
	mu_user.Unlock()
	if !ok {
		return
	}
	username := usr
	var minutos, megas int
	anio, mes, _ := time.Now().Date() //Fecha actual
	mesGrafico := fmt.Sprintf("%d-%02d", anio, mes)
	db0, err := sql.Open("sqlite3", dirMonthlys+mesGrafico+"monthly.db")
	if err != nil {
		Error.Println(err)
	}
	defer db0.Close()
	dbmon_mu.RLock()
	query, err := db0.Query("SELECT sum(minutos), sum(megabytes) FROM resumen WHERE username = ? GROUP BY username", username)
	dbmon_mu.RUnlock()
	if err != nil {
		Error.Println(err)
		return
	}
	for query.Next() {
		err = query.Scan(&minutos, &megas)
		if err != nil {
			Warning.Println(err)
			continue
		}
	}
	query.Close()
	table := fmt.Sprintf("<tr><th>Total de horas consumidas: </th><td>&nbsp;</td><td>%d</td></tr><tr><th>Total de GB consumidos: </th><td>&nbsp;</td><td>%d</td></tr>", minutos, megas)
	fmt.Fprintf(w, "%s", table)
}

// Funcion que muestra el total de horas y gigas consumidos (con cambio de mes)
func totalMonthsChange(w http.ResponseWriter, r *http.Request) {
	cookie, err3 := r.Cookie(CookieName)
	if err3 != nil {
		return
	}
	key := cookie.Value
	mu_user.Lock()
	usr, ok := user[key] // De aquí podemos recoger el usuario
	mu_user.Unlock()
	if !ok {
		return
	}
	username := usr
	r.ParseForm()
	var minutos, megas int
	mesGrafico := r.FormValue("years") + "-" + r.FormValue("months")
	//Se comprueba si existe la base de datos mensual
	if _, err := os.Stat(dirMonthlys + mesGrafico + "monthly.db"); os.IsNotExist(err) {
		//No hay base de datos
		Warning.Println("No existe el fichero de base de datos")
		Error.Println(err)
		fmt.Fprintf(w, "%s", "NoBD")
	} else {
		db0, err := sql.Open("sqlite3", dirMonthlys+mesGrafico+"monthly.db")
		if err != nil {
			Error.Println(err)
		}
		defer db0.Close()
		dbmon_mu.RLock()
		query, err := db0.Query("SELECT sum(minutos), sum(megabytes) FROM resumen WHERE username = ? GROUP BY username", username)
		dbmon_mu.RUnlock()
		if err != nil {
			Warning.Println(err)
			return
		}
		for query.Next() {
			err = query.Scan(&minutos, &megas)
			if err != nil {
				Warning.Println(err)
				continue
			}
		}
		query.Close()
		table := fmt.Sprintf("<tr><th>Total de horas consumidas: </th><td>&nbsp;</td><td>%d</td></tr><tr><th>Total de GB consumidos: </th><td>&nbsp;</td><td>%d</td></tr>", minutos, megas)
		fmt.Fprintf(w, "%s", table)
	}
}

// Se crean los canvas para colocar los gráficos
func createGraf(w http.ResponseWriter, r *http.Request) {
	canv1 := "<label>Audiencia Total por personas</label><canvas id='graficop1'/>"
	canv2 := "<label>Tiempo Total Visionado</label><canvas id='graficop2'/>"
	canv3 := "<label>Tiempo Medio visionado en horas</label><canvas id='graficop3'/>"
	canv4 := "<label>Tráfico diario en GigaBytes</label><canvas id='graficop4'/>"
	canv5 := "<label>Máximo simultaneos en personas</label><canvas id='graficop5'/>"
	canv6 := "<label>Hora Pico de Audiencia</label><canvas id='graficop6'/>"
	fmt.Fprintf(w, "%s;%s;%s;%s;%s;%s", canv1, canv2, canv3, canv4, canv5, canv6)
}

// devuelve el numero de dias de un mes y año determinados
func daysIn(m time.Month, year int) int {
	return time.Date(year, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// devuelve el numero de dias de un mes y año determinados recibiendo un string
func daysStringIn(mes string, year int) int {
	m, _ := strconv.Atoi(mes)
	return time.Date(year, time.Month(m+1), 0, 0, 0, 0, 0, time.UTC).Day()
}

// años por arriba y por abajo del actual
func UpDownYears(year int) []int {
	var array []int
	cont := 1
	cont2 := -2
	for i := 0; i < 2; i++ {
		array = append(array, year+cont2)
		cont2++
	}
	array = append(array, year)
	for i := 0; i < 2; i++ {
		array = append(array, year+cont)
		cont++
	}
	return array
}

//funcion que va a colocar las datos monthly en sus correspondientes dias
func grafDaysFloat(hora map[int]float64, day int) []float64 {
	x := make([]float64, day)
	for cont, _ := range x {
		for key, value := range hora {
			if key == cont+1 {
				x[cont] = value
			}
		}
	}
	return x
}

//funcion que va a colocar las datos monthly en sus correspondientes dias
func grafDays(hora map[int]int, day int) []int {
	x := make([]int, day)
	for cont, _ := range x {
		for key, value := range hora {
			if key == cont+1 {
				x[cont] = value
			}
		}
	}
	return x
}
