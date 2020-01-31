package main

import (
	"bufio"
	"crypto/tls"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/oschwald/geoip2-golang"
	"github.com/todostreaming/gohw"
	"github.com/todostreaming/syncmap"
	"golang.org/x/crypto/acme/autocert"
)

var (
	db         *sql.DB
	db_mu      sync.Mutex
	dbday_mu   sync.RWMutex
	dbmon_mu   sync.RWMutex
	Info       *log.Logger
	Warning    *log.Logger
	Error      *log.Logger
	Bw_int     *syncmap.SyncMap
	Hardw      *gohw.GoHw
	dbgeoip    *geoip2.Reader
	mu_dbgeoip sync.Mutex
	numgo      int //number of goroutines working
	dominio    string
	server     string
	iface      string
	ssldom     string
	restric    bool // if domain / dominio = "no" restrict = false
	onlyweb    bool
	stats      bool // true = stats ok; false = no stats at all
)

// Inicializamos la conexion a BD y el log de errores
func init() {
	var err_db error
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Fallo al abrir el archivo de error:", err)
	}
	Info = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Warning = log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(io.MultiWriter(file, os.Stderr), "ERROR :", log.Ldate|log.Ltime|log.Lshortfile)
	// Antes de abrir la BD live
	if _, err := os.Stat(DirRamDB + "live.db"); err != nil { // es la primera ejecución, o hemos reiniciado la maquina (reboot)
		exec.Command("/bin/sh", "-c", fmt.Sprintf("cp -f %slive.db* %s", DirDB, DirRamDB)).Run()
		exec.Command("/bin/sh", "-c", "sync").Run()
	}
	db, err_db = sql.Open("sqlite3", DirRamDB+"live.db")
	if err_db != nil {
		Error.Println(err_db)
		log.Fatalln("Fallo al abrir el archivo de error:", err_db)
	}
	db.Exec("PRAGMA journal_mode=WAL;")
	Bw_int = syncmap.New()

	// Antes de abrir la BD GeoIP2 City
	if _, err := os.Stat(DirRamDB + "GeoIP2-City.mmdb"); err != nil { // es la primera ejecución, o hemos reiniciado la maquina (reboot)
		exec.Command("/bin/sh", "-c", fmt.Sprintf("cp -f %sGeoIP2-City.mmdb* %s", DirDB, DirRamDB)).Run()
		exec.Command("/bin/sh", "-c", "sync").Run()
	}
	dbgeoip, err = geoip2.Open(DirRamDB + "GeoIP2-City.mmdb")
	if err != nil {
		log.Fatal("Fallo al abrir el GeoIP2:", err)
	}
}

// funcion principal del programa
func main() {
	fmt.Printf("Golang HTTP Server starting at Port %s ...\n", http_port)
	if session {
		fmt.Println("SESSION Cookies capability enabled !!!")
	} else {
		fmt.Println("SESSION Cookies capability disabled !!!")
	}

	if session { // will delete expired sessions previously recorded
		go controlinternalsessions()
	}
	loadSettings(playingsRoot)
	mu_cloud.Lock()
	dominio = cloud["domain"]
	iface = cloud["iface"]
	server = cloud["cloudserver"]
	ssldom = cloud["ssldom"]
	if cloud["stats"] == "yes" {
		stats = true
	} else {
		stats = false
	}
	if cloud["onlyweb"] == "no" {
		onlyweb = false
	} else {
		onlyweb = true
	}
	mu_cloud.Unlock()
	if dominio == "no" {
		restric = false
	} else {
		restric = true
	}
	Hardw = gohw.Hardware()
	Hardw.Run(iface)
	go func() {
		for {
			if procsrunning("nginx") == 0 {
				exec.Command("/bin/sh", "-c", "/usr/bin/nginx").Run()
			}
			time.Sleep(1 * time.Second)
		}
	}()
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			db_mu.Lock()
			exec.Command("/bin/sh", "-c", fmt.Sprintf("cp -f %slive.db* %s", DirRamDB, DirDB)).Run()
			exec.Command("/bin/sh", "-c", "sync").Run()
			db_mu.Unlock()
		}
	}()
	go func() {
		for {
			numgo = runtime.NumGoroutine()
			time.Sleep(100 * time.Millisecond)

		}
	}()
	go mantenimiento()
	go encoder()

	mux := http.NewServeMux()
	mux.HandleFunc("/", root)
	mux.HandleFunc(login_cgi, login)
	mux.HandleFunc(logout_cgi, logout)
	// Handlers de graficos
	mux.HandleFunc("/encoderStatNow.cgi", encoderStatNow)
	mux.HandleFunc("/playerStatNow.cgi", playerStatNow)
	mux.HandleFunc("/consultaFecha.cgi", consultaFecha)
	mux.HandleFunc("/firstFecha.cgi", firstFecha)
	mux.HandleFunc("/getMonthsYears.cgi", getMonthsYears)
	mux.HandleFunc("/giveFecha.cgi", giveFecha)
	mux.HandleFunc("/zeroFields.cgi", zeroFields)
	mux.HandleFunc("/formatDaylyhtml.cgi", formatDaylyhtml)
	mux.HandleFunc("/createGraf.cgi", createGraf)
	mux.HandleFunc("/firstMonthly.cgi", firstMonthly)
	mux.HandleFunc("/graficosMonthly.cgi", graficosMonthly)
	mux.HandleFunc("/play.cgi", play)
	mux.HandleFunc("/publish.cgi", publish)
	mux.HandleFunc("/onplay.cgi", onplay)
	mux.HandleFunc("/getMonthsYearsAdmin.cgi", getMonthsYearsAdmin)
	mux.HandleFunc("/putMonthlyAdmin.cgi", putMonthlyAdmin)
	mux.HandleFunc("/putMonthlyAdminChange.cgi", putMonthlyAdminChange)
	mux.HandleFunc("/editar_admin.cgi", editar_admin)
	mux.HandleFunc("/editar_cliente.cgi", editar_cliente)
	mux.HandleFunc("/user_admin.cgi", user_admin)
	mux.HandleFunc("/changeStatus.cgi", changeStatus)
	mux.HandleFunc("/nuevoCliente.cgi", nuevoCliente)
	mux.HandleFunc("/borrarCliente.cgi", borrarCliente)
	mux.HandleFunc("/buscarClientes.cgi", buscarClientes)
	mux.HandleFunc("/totalMonths.cgi", totalMonths)
	mux.HandleFunc("/totalMonthsChange.cgi", totalMonthsChange)
	mux.HandleFunc("/hardware.cgi", gethardware)

	certManager := autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache("/etc/certs"),
	}

	server := &http.Server{
		Addr:    ":443",
		Handler: mux,
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}

	go http.ListenAndServe(":80", certManager.HTTPHandler(nil))
	server.ListenAndServeTLS("", "")

	//go http.ListenAndServeTLS(":443", "/etc/letsencrypt/live/"+ssldom+"/fullchain.pem", "/etc/letsencrypt/live/"+ssldom+"/privkey.pem", nil) // Servidor HTTPS/2 multihilo
	//log.Fatal(http.ListenAndServe(":"+http_port, nil))                                                                                  // Servidor HTTP clasico
}

func redirect(w http.ResponseWriter, req *http.Request) {
	// remove/add not default ports from req.Host
	target := "https://" + req.Host + req.URL.Path
	if len(req.URL.RawQuery) > 0 {
		target += "?" + req.URL.RawQuery
	}
	log.Printf("redirect to: %s", target)
	http.Redirect(w, req, target, http.StatusTemporaryRedirect)
}

func gethardware(w http.ResponseWriter, r *http.Request) {
	st := Hardw.Status()

	var cpu, ram, cpused, ramUsed, upload, download string

	cpu = fmt.Sprintf("%s (%d cores)", st.CPUName, st.CPUCores)
	ram = fmt.Sprintf("%d MB", st.TotalMem/1024/1000)

	if st.TotalMem > 0 {
		cpused = fmt.Sprintf("%d%%", int(st.CPUusage))
		ramUsed = fmt.Sprintf("%d%%", 100*st.UsedMem/st.TotalMem)
		upload = fmt.Sprintf("%d Kbps", st.RXbps/1000)
		download = fmt.Sprintf("%d Kbps", st.TXbps/1000)
	}
	/*
		Quiero en la página una tabla con todos los datos expuestos y recargados automaticamente cada 10 segundos así:

		"CPU: %s (%d cores)", st.CPUName, st.CPUCores
		"RAM: %d MB\n", st.TotalMem/1024/1000
		"CPU used: %d%%", int(st.CPUusage)
		"RAM used: %2d%%", 100*st.UsedMem/st.TotalMem   (importante revisar que st.TotalMem > 0, o puede dar un panic por dividir por cero)
		"Upload: %d Mbps", st.RXbps/1000000
		"Download: %d Mbps", st.TXbps/1000000
	*/

	fmt.Fprintf(w, "%s;%s;%s;%s;%s;%s", cpu, ram, cpused, ramUsed, upload, download)
}

func encoder() {
	var username, streamname string
	var count int
	for {
		type Client struct {
			Ip      string `xml:"address"`
			Time    string `xml:"time"`
			Publish int    `xml:"publishing"`
		}
		type Stream struct {
			Nombre     string   `xml:"name"`
			Bw_in      string   `xml:"bw_in"`
			Width      string   `xml:"meta>video>width"`
			Height     string   `xml:"meta>video>height"`
			Frame      string   `xml:"meta>video>frame_rate"`
			Vcodec     string   `xml:"meta>video>codec"`
			Acodec     string   `xml:"meta>audio>codec"`
			ClientList []Client `xml:"client"`
		}
		type Result struct {
			Stream []Stream `xml:"server>application>live>stream"`
		}
		resp, err := http.Get("http://127.0.0.1:8080/stats")
		if err != nil {
			Warning.Println(err)
			time.Sleep(3 * time.Second)
			continue
		}
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			Warning.Println(err)
			time.Sleep(3 * time.Second)
			continue
		}
		v := Result{}
		err = xml.Unmarshal([]byte(body), &v)
		if err != nil {
			Error.Printf("xml read error: %s", err)
			time.Sleep(3 * time.Second)
			continue
		}
		for _, val := range v.Stream {
			for _, val2 := range val.ClientList {
				if val2.Publish == 1 {
					userstream := strings.Split(val.Nombre, "-")
					if len(userstream) > 1 {
						username = userstream[0]
						streamname = userstream[1]
					}
					tiempo := toInt(val2.Time) / 1000        // Conversión msec to sec
					tiempo_now := time.Now().Unix()          // Tiempo actual
					Bw_int.Set(val.Nombre, toInt(val.Bw_in)) // Guardamos el bitrate
					info := fmt.Sprintf("%sx%sx%s %s/%s", val.Width, val.Height, val.Frame, val.Vcodec, val.Acodec)
					db_mu.Lock()
					err := db.QueryRow("SELECT count(*) FROM encoders WHERE username = ? AND streamname = ? AND ip= ?", username, streamname, val2.Ip).Scan(&count)
					db_mu.Unlock()
					if err != nil {
						Error.Println(err)
						continue
					}
					//Cuando no existe usuario, stream e ip
					if count == 0 {
						city, region, country, isocode, timezone, lat, long := geoIP(val2.Ip) // Datos de geolocalización
						if isocode == "" {
							isocode = "OT" //cuando el isocode esta vacio, lo establecemos a OT (other)
						}
						if country == "" {
							country = "Unknown" //cuando el country esta vacio, lo establecemos a Unknown (desconocido)
						}
						db_mu.Lock()
						_, err1 := db.Exec("INSERT INTO encoders (`username`, `streamname`, `time`, `bitrate`, `ip`, `info`, `isocode`, `country`, `region`, `city`, `timezone`, `lat`, `long`, `timestamp`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
							username, streamname, tiempo, toInt(val.Bw_in), val2.Ip, info, isocode, country, region, city, timezone, lat, long, tiempo_now)
						db_mu.Unlock()
						if err1 != nil {
							Error.Println(err1)
						}
					} else {
						db_mu.Lock()
						_, err1 := db.Exec("UPDATE encoders SET username=?, streamname=?, time=?, bitrate=?, ip=?, info=?, timestamp=? WHERE username = ? AND streamname = ? AND ip = ?",
							username, streamname, tiempo, toInt(val.Bw_in), val2.Ip, info, tiempo_now, username, streamname, val2.Ip)
						db_mu.Unlock()
						if err1 != nil {
							Error.Println(err1)
						}
					}
				}
			}
		}
		time.Sleep(3 * time.Second)
	}
}

// TAREAS DE MANTENIMIENTO
func mantenimiento() {
	var fecha_actual, fecha_antigua string
	var mes_actual, mes_antiguo string
	for {
		cambio_de_fecha := false
		cambio_de_mes := false
		hh, mm, _ := time.Now().Clock()
		anio, mes, dia := time.Now().Date() //Fecha actual
		// Se saca la hora y los minutos
		fecha_actual = fmt.Sprintf("%04d-%02d-%02d", anio, mes, dia) // Calculo de fecha actual
		// Se comprueba si hay cambio de dia
		if fecha_actual != fecha_antigua { // dayly.db
			cambio_de_fecha = true
			if _, err := os.Stat(dirDaylys + fecha_actual + "dayly.db"); err == nil {
				cambio_de_fecha = false // se debe a un reinicio del hlserver
			}
		}
		// Se comprueba si hay cambio de mes
		mes_actual = fecha_actual[0:7] // year-month
		if mes_actual != mes_antiguo { // monthly.db
			cambio_de_mes = true
			if _, err := os.Stat(dirMonthlys + mes_actual + "monthly.db"); err == nil {
				cambio_de_mes = false // se debe a un reinicio del hlserver
			}
		}
		if cambio_de_mes {
			// Aqui hago la copia de monthly.db en mes_actual + monthly.db
			exec.Command("/bin/sh", "-c", "cp "+monthlyDB+" "+dirMonthlys+mes_actual+"monthly.db").Run()
		}
		if cambio_de_fecha {
			//Comprobamos si existe el fichero con fecha antigua
			if _, err := os.Stat(dirDaylys + fecha_antigua + "dayly.db"); os.IsNotExist(err) {
				// Aqui hago la copia de dayly.db en fecha_actual + dayly.db
				exec.Command("/bin/sh", "-c", "cp "+daylyDB+" "+dirDaylys+fecha_actual+"dayly.db").Run()
			} else {
				exec.Command("/bin/sh", "-c", "cp "+daylyDB+" "+dirDaylys+fecha_actual+"dayly.db").Run()
				limit_time := time.Now().Unix() - 86400
				//Sacamos los datos de la fecha
				datos_antiguos := strings.Split(fecha_antigua, "-")
				fechaMonth := fmt.Sprintf("%s:%s", datos_antiguos[1], datos_antiguos[2])
				// Antes de nada borramos los players con timestamp a más de 1 día
				db_mu.Lock()
				db.Exec("DELETE FROM players WHERE timestamp < ?", limit_time)
				db_mu.Unlock()
				// Se seleccionan el total de Ips, las horas totales y el total de Gigabytes
				db_mu.Lock()
				query, err := db.Query("SELECT count(ipclient), sum(total_time)/3600, sum(kilobytes)/1000000, username, streamname FROM players GROUP BY username, streamname")
				db_mu.Unlock()
				if err != nil {
					Error.Println(err)
					continue
				}
				db1, err := sql.Open("sqlite3", dirDaylys+fecha_antigua+"dayly.db") // Apertura de la dateDayly.db antigua para lectura del pico/hora
				if err != nil {
					Error.Println(err)
					continue
				}
				db2, err := sql.Open("sqlite3", dirMonthlys+mes_antiguo+"monthly.db") // Apertura de mes actual + Monthly.db para escritura del resumen del pasado dia
				if err != nil {
					Error.Println(err)
					continue
				}
				//Declaracion de variables
				var ips, horas, gigas, pico, horapico, minpico int
				var userName, streamName string
				for query.Next() {
					err = query.Scan(&ips, &horas, &gigas, &userName, &streamName)
					if err != nil {
						Error.Println(err)
						continue
					}
					// Se seleccionan el máximo de usuarios conectados, y la hora:min de la dayly antigua
					// SELECT sum(count) AS cuenta, username, streamname, hour, minutes FROM resumen WHERE username = ? AND streamname = ? GROUP BY username, streamname, hour, minutes ORDER BY cuenta DESC
					dbday_mu.RLock()
					err := db1.QueryRow("SELECT sum(count) AS cuenta, username, streamname, hour, minutes FROM resumen WHERE username = ? AND streamname = ? GROUP BY username, streamname, hour, minutes ORDER BY cuenta DESC", userName, streamName).Scan(&pico, &userName, &streamName, &horapico, &minpico)
					dbday_mu.RUnlock()
					if err != nil {
						Error.Println(err)
						continue
					}
					hourMin := fmt.Sprintf("%02d:%02d", horapico, minpico) //hour:min para monthly.db
					dbmon_mu.Lock()
					// Inserto los datos de resumen mensual
					_, err1 := db2.Exec("INSERT INTO resumen (`username`,`streamname`, `audiencia`, `minutos`, `pico`, `horapico`, `megabytes`, `fecha`) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
						userName, streamName, ips, horas, pico, hourMin, gigas, fechaMonth)
					dbmon_mu.Unlock()
					if err1 != nil {
						Error.Println(err1)
					}
				}
				query.Close()
				db2.Close()
				db1.Close()
				// Ponemos kilobytes, total_time a CERO de live.db xq empezamos un nuevo dia con trafico y horas acumuladas a CERO
				db_mu.Lock()
				db.Exec("UPDATE players SET kilobytes=0 , total_time=0")
				db_mu.Unlock()
			}
		}
		// Solo grabaremos en este minuto en dayly.db los q estan activos ahora mismo
		tiempo_limite := time.Now().Unix() - 30
		var user, stream, so, isocode string
		var num_filas, total_time, total_kb int
		db3, err := sql.Open("sqlite3", dirDaylys+fecha_actual+"dayly.db") // Apertura de dateDayly.db
		if err != nil {
			Error.Println(err)
		}
		db_mu.Lock()
		query, err := db.Query("SELECT count(ipclient), username, streamname, os,  isocode, sum(total_time), sum(kilobytes) FROM players WHERE timestamp > ? AND time > 0 GROUP BY username, streamname, os, isocode", tiempo_limite)
		db_mu.Unlock()
		if err != nil {
			Error.Println(err)
			continue
		}
		for query.Next() {
			err = query.Scan(&num_filas, &user, &stream, &so, &isocode, &total_time, &total_kb)
			if err != nil {
				Error.Println(err)
				continue
			}
			dbday_mu.Lock()
			// inserto los datos de resumen
			_, err1 := db3.Exec("INSERT INTO resumen (`username`, `streamname`, `os`, `isocode`, `time`, `kilobytes`, `count`, `hour`, `minutes`, `date`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
				user, stream, so, isocode, total_time, total_kb, num_filas, hh, mm, fecha_actual)
			dbday_mu.Unlock()
			if err1 != nil {
				Error.Println(err1)
			}
		}
		query.Close()
		db3.Close()

		fecha_antigua = fecha_actual
		mes_antiguo = mes_actual
		time.Sleep(1 * time.Minute)
	}
}

func geoIP(ip_parsing string) (city, region, country, isocode, timezone string, lat, long float64) {
	// If you are using strings that may be invalid, check that ip is not nil
	ip := net.ParseIP(ip_parsing)
	record, err := dbgeoip.City(ip)
	if err != nil {
		return
	}
	city = record.City.Names["en"]
	if len(record.Subdivisions) > 0 {
		region = record.Subdivisions[0].Names["en"]
	}
	country = record.Country.Names["en"]
	isocode = record.Country.IsoCode
	timezone = record.Location.TimeZone
	lat = record.Location.Latitude
	long = record.Location.Longitude

	return city, region, country, isocode, timezone, lat, long
}

func loadSettings(filename string) {
	fr, err := os.Open(filename)
	defer fr.Close()
	if err == nil {
		reader := bufio.NewReader(fr)
		for {
			linea, rerr := reader.ReadString('\n')
			if rerr != nil {
				break
			}
			if strings.Contains(linea, "#") {
				continue
			}
			linea = strings.TrimRight(linea, "\n")
			item := strings.Split(linea, " = ")
			mu_cloud.Lock()
			if len(item) == 2 {
				cloud[item[0]] = item[1]
			}
			mu_cloud.Unlock()
		}
	}
}

//ver si un proceso está corriendo
func procsrunning(name string) int {
	exe := fmt.Sprintf("/usr/bin/pgrep %s | /usr/bin/wc -l", name)
	out, _ := exec.Command("/bin/sh", "-c", exe).CombinedOutput()
	num, _ := strconv.Atoi(strings.TrimRight(string(out), "\n"))
	return num
}

func secs2time(seconds int) (time string) {
	horas := int(seconds / 3600)
	minutos := int((seconds - (horas * 3600)) / 60)
	segundos := seconds - (horas * 3600) - (minutos * 60)
	time = fmt.Sprintf("%dh:%02dm:%02ds", horas, minutos, segundos)

	return
}
