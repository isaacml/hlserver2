package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func getMonthsYearsAdmin(w http.ResponseWriter, r *http.Request) {
	var menu1, menu2 string
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
	fmt.Fprintf(w, "%s;%s", menu1, menu2)
}

// Funcion que muestra los datos mensuales de los clientes
func putMonthlyAdmin(w http.ResponseWriter, r *http.Request) {
	anio, mes, _ := time.Now().Date() //Fecha actual
	table := "<tr><th>Usuario</th><th>Contraseña</th><th>Horas</th><th>Gigabytes</th><th>Status</th></tr>"
	mesGrafico := fmt.Sprintf("%d-%02d", anio, mes)
	db0, err := sql.Open("sqlite3", dirMonthlys+mesGrafico+"monthly.db")
	if err != nil {
		Error.Println(err)
	}
	defer db0.Close()
	db_mu.Lock()
	query2, err := db.Query("SELECT id, username, password, status FROM admin WHERE type = 0")
	db_mu.Unlock()
	if err != nil {
		Error.Println(err)
		return
	}
	for query2.Next() {
		var user, pass, estado string
		var id, status, minutos, megas int
		err = query2.Scan(&id, &user, &pass, &status)
		if status == 1 {
			estado = "ON"
		} else {
			estado = "OFF"
		}
		dbmon_mu.RLock()
		query, err := db0.Query("SELECT sum(minutos), sum(megabytes) FROM resumen WHERE username = ? GROUP BY username", user)
		dbmon_mu.RUnlock()
		if err != nil {
			Error.Println(err)
		}
		for query.Next() {
			err = query.Scan(&minutos, &megas)
			if err != nil {
				Warning.Println(err)
			}
		}
		query.Close()
		table += fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%d</td><td>%d</td><td><button href='#' title='Pulsa para cambiar el estado' onclick='load(%d)'>%s</button></td></tr>", user, pass, minutos, megas, id, estado)
	}
	query2.Close()
	fmt.Fprintf(w, "%s", table)
}

// Funcion que muestra los datos mensuales de los clientes
func putMonthlyAdminChange(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	table := "<tr><th>Usuario</th><th>Contraseña</th><th>Horas</th><th>Gigabytes</th><th>Status</th></tr>"
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
		db_mu.Lock()
		query2, err := db.Query("SELECT id, username, password, status FROM admin WHERE type = 0")
		db_mu.Unlock()
		if err != nil {
			Error.Println(err)
			return
		}
		for query2.Next() {
			var user, pass, estado string
			var id, status, minutos, megas int
			err = query2.Scan(&id, &user, &pass, &status)
			if status == 1 {
				estado = "ON"
			} else {
				estado = "OFF"
			}
			dbmon_mu.RLock()
			query, err := db0.Query("SELECT sum(minutos), sum(megabytes) FROM resumen WHERE username = ? GROUP BY username", user)
			dbmon_mu.RUnlock()
			if err != nil {
				Error.Println(err)
			}
			for query.Next() {
				err = query.Scan(&minutos, &megas)
				if err != nil {
					Warning.Println(err)
				}
			}
			query.Close()
			table += fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%d</td><td>%d</td><td><button href='#' title='Pulsa para cambiar el estado' onclick='load(%d)'>%s</button></td></tr>", user, pass, minutos, megas, id, estado)
		}
		query2.Close()
		fmt.Fprintf(w, "%s", table)
	}
}

// Funcion cambia el estado ON/OFF a los clientes
func changeStatus(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var id, status int
	var user string
	db_mu.Lock()
	query2, err := db.Query("SELECT id, username, status FROM admin WHERE id = ?", r.FormValue("load"))
	db_mu.Unlock()
	if err != nil {
		Error.Println(err)
		return
	}
	for query2.Next() {
		err = query2.Scan(&id, &user, &status)
	}
	query2.Close()
	if status == 1 {
		db_mu.Lock()
		_, err1 := db.Exec("UPDATE admin SET status = 0 WHERE id= ?", id)
		db_mu.Unlock()
		if err1 != nil {
			Error.Println(err1)
		}
		time.Sleep(10 * time.Millisecond)
		// Seleccionamos todos los streams pertenecientes a un usuario, para hecharlos fuera
		db_mu.Lock()
		query3, err := db.Query("SELECT streamname FROM encoders WHERE username = ?", user)
		db_mu.Unlock()
		if err != nil {
			Error.Println(err)
			return
		}
		for query3.Next() {
			var streams string
			err = query3.Scan(&streams)
			nombre := fmt.Sprintf("%s-%s", user, streams)
			//Sacamos uno a uno los streams
			peticion := fmt.Sprintf("http://127.0.0.1:8080/control/drop/publisher?app=live&name=%s", nombre)
			http.Get(peticion)
			time.Sleep(10 * time.Millisecond)
		}
		query3.Close()
	} else {
		db_mu.Lock()
		_, err1 := db.Exec("UPDATE admin SET status = 1 WHERE id= ?", id)
		db_mu.Unlock()
		if err1 != nil {
			Error.Println(err1)
		}
	}
}

// Función para dar de alta un nuevo cliente en la tabla admin
func nuevoCliente(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	db_mu.Lock()
	_, err1 := db.Exec("INSERT INTO admin (`username`, `password`, `type`, `status`) VALUES (?, ?, ?, ?)",
		r.FormValue("nom_cli"), r.FormValue("passw"), r.FormValue("type"), r.FormValue("status"))
	db_mu.Unlock()
	if err1 != nil {
		Error.Println(err1)
	}
}

// Función que selecciona los clientes de la tabla admin
func buscarClientes(w http.ResponseWriter, r *http.Request) {
	var id int
	var nombre, selector string
	db_mu.Lock()
	query, err := db.Query("SELECT id, username FROM admin WHERE type = 0")
	db_mu.Unlock()
	if err != nil {
		Error.Println(err)
		return
	}
	for query.Next() {
		err = query.Scan(&id, &nombre)
		if err != nil {
			Warning.Println(err)
		}
		selector = fmt.Sprintf("<option value='%d'>%s</option>", id, nombre)
		fmt.Fprintf(w, "%s", selector)
	}
	query.Close()
}

// Función que borra un cliente de la tabla admin
func borrarCliente(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	db_mu.Lock()
	_, err1 := db.Exec("DELETE FROM admin WHERE id = ?", r.FormValue("clients"))
	db_mu.Unlock()
	if err1 != nil {
		Error.Println(err1)
	}
}
