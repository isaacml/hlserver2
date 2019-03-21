package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/todostreaming/realip"
)

// sirve todos los ficheros estáticos de la web html,css,js,graficos,etc
func root(w http.ResponseWriter, r *http.Request) {
	var namefile string
	if onlyweb {
		if !strings.Contains(r.UserAgent(), "Mozilla") && !strings.Contains(r.UserAgent(), "stagefright") && !strings.Contains(r.UserAgent(), "Mac") {
			http.NotFound(w, r)
			return
		}
	}
	namefile = strings.TrimRight(rootdir+r.URL.Path[1:], "/")
	fileinfo, err := os.Stat(namefile)
	if err != nil {
		// fichero no existe
		http.NotFound(w, r)
		return
	} else if fileinfo.IsDir() {
		// es un directorio, luego le añadimos index.html
		namefile = namefile + "/" + first_page + ".html"
		_, err2 := os.Stat(namefile)
		if err2 != nil {
			http.NotFound(w, r)
			return
		}
	}
	fr, errn := os.Open(namefile)
	if errn != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}
	//hh, mm, ss := time.Now().Clock()
	defer fr.Close()
	if strings.Contains(namefile, ".m3u8") {
		if restric {
			u, err := url.Parse(r.Referer())
			if err != nil {
				http.NotFound(w, r)
				return
			}
			if !strings.Contains(dominio, u.Hostname()) && !strings.Contains(r.Referer(), server) { // it is not our playtv
				http.NotFound(w, r)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Expose-Headers", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileinfo.Size()))
		w.Header().Set("Accept-Ranges", "bytes")
		query, _ := url.ParseQuery(r.URL.RawQuery)
		if (numgo < 30000) && stats { // if there are more than 30k goroutines working, live stats will stop for a while
			go createStats(namefile, r.Header.Get("User-Agent"), realip.RealIP(r), getip(r.RemoteAddr), query.Get("city")) // try to use less internal variables to save ram usage
		}
		io.Copy(w, fr)
		return
	} else if strings.Contains(namefile, ".ts") {
		w.Header().Set("Cache-Control", "max-age=300")
		w.Header().Set("Content-Type", "video/MP2T")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Expose-Headers", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileinfo.Size()))
		w.Header().Set("Accept-Ranges", "bytes")
		io.Copy(w, fr)
		return
	}
	if session {
		// ?err parsing
		if strings.Contains(r.URL.String(), "?err") {
			// sustituimos <span id="loginerr"></span> por un texto de error a mostrar
			buf, _ := ioutil.ReadAll(fr) // leemos el HTML template de una sola vez
			html := string(buf)
			// Vamos a meter los campos options creados en bmdinfo() en el HTML
			html = strings.Replace(html, spanHTMLlogerr, ErrorText, -1)
			w.Header().Set("Content-Type", mime.TypeByExtension(".html"))
			fmt.Fprint(w, html)
		} else {
			// Revisar cookies
			file := strings.Split(namefile, ".")

			if (file[1] != "html") || (file[0] == (rootdir + first_page)) {
				http.ServeContent(w, r, namefile, fileinfo.ModTime(), fr)
			} else {
				cookie, err3 := r.Cookie(CookieName)
				if err3 != nil {
					Error.Println("No existe esa cookie")
					http.Redirect(w, r, "/"+first_page+".html", http.StatusFound)

				} else {
					key := cookie.Value
					mu_user.Lock()
					_, ok := user[key] // De aquí podemos recoger el usuario
					mu_user.Unlock()
					if ok {
						cookie.Expires = time.Now().Add(time.Duration(session_timeout) * time.Second)
						http.SetCookie(w, cookie)
						mu_user.Lock()
						tiempo[cookie.Value] = cookie.Expires
						mu_user.Unlock()
						http.ServeContent(w, r, namefile, fileinfo.ModTime(), fr)
					} else {
						Error.Println("No existe cookie")
						http.Redirect(w, r, "/"+first_page+".html", http.StatusFound)
					}
				}
			}
		}
	} else {
		http.ServeContent(w, r, namefile, fileinfo.ModTime(), fr)
	}
}

/*
Base de datos			Variable_OLD			Variable_NOW
=================================================================
	timestamp			time_old				time_now
	time				tiempo_old				time_connect
	totaltime			total_time_old			total_time
	kilobytes			kb_old					kilobytes

*/
func createStats(namefile, agent, forwarded, remoteip, ciudad string) {
	userAgent := map[string]string{"win": "Windows", "mac": "Mac OS X", "and": "Android", "lin": "Linux"}
	var existe bool
	var streamer, ipcliente, ipproxy, so, user, streamname string
	var time_old, time_now, time_connect, tiempo_old, kilobytes, kb_old, total_time_old, total_time int64
	//operaciones sobre el namefile
	fmt.Sscanf(namefile, "/var/segments/live/%s", &streamer)
	nom := strings.Split(streamer, ".")
	userstream := nom[0] // user-stream
	username := strings.Split(userstream, "-")
	if len(username) > 1 {
		user = username[0]       // user
		streamname = username[1] // stream
	}
	time_now = time.Now().Unix() //tiempo actual
	//operaciones para el user agent
	for key, value := range userAgent {
		if strings.Contains(agent, value) {
			so = key
			existe = true
			break
		}
	}
	//Agent User not find
	if !existe {
		so = "other"
	}
	//Cuando el forwarded está vacio
	if forwarded == "" {
		ipcliente = remoteip
		ipproxy = ""
	} else {
		ipcliente = forwarded
		ipproxy = remoteip
	}
	db_mu.Lock()
	query, err := db.Query("SELECT timestamp, time, kilobytes, total_time FROM players WHERE username = ? AND streamname = ? AND ipclient= ? AND os = ?", user, streamname, ipcliente, so)
	db_mu.Unlock()
	if err != nil {
		Error.Println(err)
		return
	}
	count := 0
	for query.Next() {
		err = query.Scan(&time_old, &tiempo_old, &kb_old, &total_time_old)
		if err != nil {
			Error.Println(err)
			continue
		}
		count++
	}
	query.Close()
	//Cuando no existe usuario, stream e ip
	if count == 0 {
		city, region, country, isocode, timezone, lat, long := geoIP(ipcliente) //Datos de geolocalización
		if ciudad != "" {
			city = ciudad
		}
		time_connect = 0
		kilobytes = 0
		total_time = 0
		if isocode == "" {
			isocode = "OT" //cuando el isocode esta vacio, lo establecemos a OT (other)
		}
		if country == "" {
			country = "Unknown" //cuando el country esta vacio, lo establecemos a Unknown (desconocido)
		}
		db_mu.Lock()
		_, err1 := db.Exec("INSERT INTO players (`username`, `streamname`, `os`, `ipproxy`, `ipclient`, `isocode`, `country`, `region`, `city`, `timezone`, `lat`, `long`, `timestamp`, `time`, `kilobytes`, `total_time`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			user, streamname, so, ipproxy, ipcliente, isocode, country, region, city, timezone, lat, long, time_now, time_connect, kilobytes, total_time)
		db_mu.Unlock()
		if err1 != nil {
			Error.Println(err1)
		}
	} else {
		v, ok := Bw_int.Get(userstream) // obtenemos una interface{}
		if ok == false {
			v = 0
		}
		bitrate := int64(v.(int) / 8000) // convertimos el valor a entero y calculamos el bitrate
		if time_now-time_old > 30 {      // desconexión a los 30"
			time_connect = 0
			total_time = total_time_old
			kilobytes = kb_old
		} else {
			time_connect = tiempo_old + (time_now - time_old) // cálculo del tiempo que lleva conectado
			total_time = total_time_old + (time_now - time_old)
			kilobytes = kb_old + (time_now-time_old)*bitrate
		}
		if ciudad != "" {
			db_mu.Lock()
			_, err1 := db.Exec("UPDATE players SET username=?, streamname=?, os=?, ipproxy=?, ipclient=?, city=?, timestamp=?, time=?, kilobytes=?, total_time=? WHERE username = ? AND streamname = ? AND ipclient = ? AND os = ?",
				user, streamname, so, ipproxy, ipcliente, ciudad, time_now, time_connect, kilobytes, total_time, user, streamname, ipcliente, so)
			db_mu.Unlock()
			if err1 != nil {
				Error.Println(err1)
			}
		} else {
			db_mu.Lock()
			_, err1 := db.Exec("UPDATE players SET username=?, streamname=?, os=?, ipproxy=?, ipclient=?, timestamp=?, time=?, kilobytes=?, total_time=? WHERE username = ? AND streamname = ? AND ipclient = ? AND os = ?",
				user, streamname, so, ipproxy, ipcliente, time_now, time_connect, kilobytes, total_time, user, streamname, ipcliente, so)
			db_mu.Unlock()
			if err1 != nil {
				Error.Println(err1)
			}
		}

	}
}
