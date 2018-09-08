package main

import (
	"fmt"
	"net/http"
)

//Función que muestra el usuario en activo
func user_admin(w http.ResponseWriter, r *http.Request) {
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
	fmt.Fprintf(w, "<input class='form-control' placeholder='Usuario' readonly='readonly' name='username' type='username' value='%s' autofocus>", username)
}

//Funcion para editar los datos del admin
func editar_admin(w http.ResponseWriter, r *http.Request) {
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
	//Solo si las contraseñas son iguales modificamos
	if r.FormValue("password") == r.FormValue("repeat-password") {
		good := "Datos modificados correctamente"
		db_mu.Lock()
		_, err1 := db.Exec("UPDATE admin SET username=?, password=? WHERE username = ? AND type = 1", r.FormValue("username"), r.FormValue("password"), username)
		db_mu.Unlock()
		if err1 != nil {
			Error.Println(err1)
		} else {
			fmt.Fprintf(w, "<div class='form-group text-success'>%s</div>", good)
		}
	} else {
		bad := "Las contraseñas no coinciden."
		fmt.Fprintf(w, "<div class='form-group text-danger'>%s</div>", bad)
	}
}

//Funcion para editar los datos del admin
func editar_cliente(w http.ResponseWriter, r *http.Request) {
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
	//Solo si las contraseñas son iguales modificamos
	if r.FormValue("password") == r.FormValue("repeat-password") {
		good := "Datos modificados correctamente"
		db_mu.Lock()
		_, err1 := db.Exec("UPDATE admin SET username=?, password=? WHERE username = ? AND type = 0", r.FormValue("username"), r.FormValue("password"), username)
		db_mu.Unlock()
		if err1 != nil {
			Error.Println(err1)
		} else {
			fmt.Fprintf(w, "<div class='form-group text-success'>%s</div>", good)
		}
	} else {
		bad := "Las contraseñas no coinciden."
		fmt.Fprintf(w, "<div class='form-group text-danger'>%s</div>", bad)
	}
}
