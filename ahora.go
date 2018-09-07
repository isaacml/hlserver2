package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

var cloud map[string]string = make(map[string]string)
var mu_cloud sync.Mutex

func encoderStatNow(w http.ResponseWriter, r *http.Request) {

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
	anio, mes, dia := time.Now().Date()
	fecha := fmt.Sprintf("%02d/%02d/%02d", dia, mes, anio)
	hh, mm, _ := time.Now().Clock()
	hora := fmt.Sprintf("%02d:%02d", hh, mm)
	tiempo_limite := time.Now().Unix() - 6 //tiempo limite de 6 seg
	db_mu.Lock()
	query, err := db.Query("SELECT streamname, isocode, ip, country, time, bitrate, info FROM encoders WHERE username = ? AND timestamp > ?", username, tiempo_limite)
	db_mu.Unlock()
	if err != nil {
		Error.Println(err)
	}
	fmt.Fprintf(w, "<h1>%s</h1><p><b>Conectados el día %s a las %s UTC</b></p><table class=\"table table-striped table-bordered table-hover\"><th>Play</th><th>INFO</th><th>País</th><th>IP</th><th>Stream</th><th>Tiempo conectado</th>", username, fecha, hora)
	for query.Next() {
		var isocode, country, streamname, ip, time_connect, info string
		var tiempo, bitrate int
		err = query.Scan(&streamname, &isocode, &ip, &country, &tiempo, &bitrate, &info)
		if err != nil {
			Warning.Println(err)
		}
		isocode = strings.ToLower(isocode)
		time_connect = secs2time(tiempo)
		INFO := fmt.Sprintf("%s [%d kbps]", info, bitrate/1000)
		fmt.Fprintf(w, "<tr><td><a href=\"javascript:launchRemote('play.cgi?stream=%s')\"><img src='images/play.jpg' border='0' title='Play %s'/></a></td><td>%s</td><td><img src=\"images/flags/%s.png\" title=\"%s\"></td><td>%s</td><td>%s</td><td>%s</td></tr>",
			streamname, streamname, INFO, isocode, country, ip, streamname, time_connect)
	}
	query.Close()
	fmt.Fprintf(w, "</table>")
}

func playerStatNow(w http.ResponseWriter, r *http.Request) {
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
	var contador int
	tiempo_limite := time.Now().Unix() - 30 //tiempo limite de 30 seg
	db_mu.Lock()
	err := db.QueryRow("SELECT count(*) FROM players WHERE username = ? AND timestamp > ? AND time > 0", username, tiempo_limite).Scan(&contador)
	db_mu.Unlock()
	if err != nil {
		Error.Println(err)
	}
	if contador >= 100 {
		db_mu.Lock()
		query, err := db.Query("SELECT isocode, country, count(ipclient) AS count, streamname FROM players WHERE username = ? AND timestamp > ? AND time > 0 GROUP BY isocode, streamname ORDER BY streamname, count DESC", username, tiempo_limite)
		db_mu.Unlock()
		if err != nil {
			Error.Println(err)
		}
		fmt.Fprintf(w, "<table class=\"table table-striped table-bordered table-hover\"><th>País</th><th>Cantidad de IPs</th><th>Stream</th>")
		fmt.Fprintf(w, "<tr><td align=\"center\" colspan='3'><b>Total:</b> %d players conectados</td></tr>", contador)
		for query.Next() {
			var isocode, country, ips, streamname string
			err = query.Scan(&isocode, &country, &ips, &streamname)
			if err != nil {
				Warning.Println(err)
			}
			isocode = strings.ToLower(isocode)
			fmt.Fprintf(w, "<tr><td>%s <img class='pull-right' src=\"images/flags/%s.png\" title=\"%s\"></td><td>%s</td><td>%s</td></tr>",
				country, isocode, country, ips, streamname)
		}
		query.Close()
		fmt.Fprintf(w, "</table>")
	} else {
		db_mu.Lock()
		query, err := db.Query("SELECT isocode, country, region, city, ipclient, os, streamname, time FROM players WHERE username = ? AND timestamp > ? AND time > 0 ORDER BY streamname, time DESC", username, tiempo_limite)
		db_mu.Unlock()
		if err != nil {
			Warning.Println(err)
		}
		fmt.Fprintf(w, "<table class=\"table table-striped table-bordered table-hover\"><th>País</th><th>Region</th><th>Ciudad</th><th>Dirección IP</th><th>Stream</th><th>O.S</th><th>Tiempo conectado</th>")
		fmt.Fprintf(w, "<tr><td align=\"center\" colspan='7'><b>Total:</b> %d players conectados</td></tr>", contador)
		for query.Next() {
			var isocode, country, region, city, ipclient, os, streamname, time_connect string
			var tiempo int
			err = query.Scan(&isocode, &country, &region, &city, &ipclient, &os, &streamname, &tiempo)
			if err != nil {
				Warning.Println(err)
			}
			isocode = strings.ToLower(isocode)
			time_connect = secs2time(tiempo)
			fmt.Fprintf(w, "<tr><td>%s <img class='pull-right' src=\"images/flags/%s.png\" title=\"%s\"></td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>",
				country, isocode, country, region, city, ipclient, streamname, os, time_connect)
		}
		query.Close()
		fmt.Fprintf(w, "</table>")
	}
}

func play(w http.ResponseWriter, r *http.Request) {
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
	loadSettings(playingsRoot)
	r.ParseForm() // recupera campos del form tanto GET como POST
	allname := username + "-" + r.FormValue("stream")
	mu_cloud.Lock()
	stream := cloud["proto"] + "://" + cloud["cloudserver"] + "/live/" + allname + ".m3u8"
	mu_cloud.Unlock()
	//video := fmt.Sprintf("<script type='text/javascript' src='http://www.domainplayers.org/js/jwplayer.js'></script><div id='container'><video width='600' height='409' controls autoplay src='%s'/></div><script type='text/javascript'>jwplayer('container').setup({ width: '600', height: '409', skin: 'http://www.domainplayers.org/newtubedark.zip', plugins: { 'http://www.domainplayers.org/qualitymonitor.swf' : {} }, image: '', modes: [{ type:'flash', src:'http://www.domainplayers.org/player.swf', config: { autostart: 'true', provider:'http://www.domainplayers.org/HLSProvider5.swf', file:'%s' } }]});</script>", stream, stream)
	video := fmt.Sprintf("<script src=\"./hls.min.js\"></script><script src=\"./html5play.min.js\"></script><video id=\"video_x890\" controls width=\"600\" height=\"409\"><source id=\"src_x890\">Your browser does not support HTML5 video. We recommend using <a href=\"https://www.google.es/chrome/browser/desktop/\">Google Chrome</a></video><script>var url = \"%s\";html5player(url, 1, \"video_x890\", \"src_x890\");</script>", stream)
	fmt.Fprintf(w, "%s", video)
}
